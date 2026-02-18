// Copyright observIQ, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package observiq

import (
	"errors"
	"sync"
	"sync/atomic"
	"time"

	"github.com/golang/snappy"
	"github.com/observiq/bindplane-otel-collector/internal/measurements"
	"github.com/open-telemetry/opamp-go/client"
	"github.com/open-telemetry/opamp-go/client/types"
	"github.com/open-telemetry/opamp-go/protobufs"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.uber.org/zap"
)

const maxSendRetries = 3

// MeasurementsReporter represents an object that reports throughput measurements as OTLP.
type MeasurementsReporter interface {
	OTLPMeasurements(extraAttributes map[string]string) pmetric.Metrics
}

// measurementsSender is a struct that handles periodically sending measurements via a custom message to an OpAMP endpoint.
type measurementsSender struct {
	logger          *zap.Logger
	reporter        MeasurementsReporter
	opampClient     client.OpAMPClient
	interval        time.Duration
	extraAttributes map[string]string

	changeIntervalChan   chan time.Duration
	changeAttributesChan chan map[string]string

	mux       *sync.Mutex
	isRunning bool
	done      chan struct{}
	wg        *sync.WaitGroup

	lastReportedSequence atomic.Int64
	lastSendMux          *sync.RWMutex
}

func newMeasurementsSender(l *zap.Logger, reporter MeasurementsReporter, opampClient client.OpAMPClient, interval time.Duration, extraAttributes map[string]string) *measurementsSender {
	return &measurementsSender{
		logger:          l,
		reporter:        reporter,
		opampClient:     opampClient,
		interval:        interval,
		extraAttributes: extraAttributes,

		changeIntervalChan:   make(chan time.Duration, 1),
		changeAttributesChan: make(chan map[string]string, 1),
		mux:                  &sync.Mutex{},
		isRunning:            false,
		done:                 make(chan struct{}),
		wg:                   &sync.WaitGroup{},
		lastReportedSequence: atomic.Int64{}, // Set to zero time to indicate that no metrics have been reported yet
		lastSendMux:          &sync.RWMutex{},
	}
}

// Start starts the sender. It may be called multiple times, even if the sender is already started.
func (m *measurementsSender) Start() {
	m.mux.Lock()
	defer m.mux.Unlock()

	if m.isRunning {
		return
	}

	m.isRunning = true

	m.wg.Add(1)
	go func() {
		defer m.wg.Done()
		m.loop()
	}()
}

// SetInterval changes the interval of the measurements sender.
func (m *measurementsSender) SetInterval(d time.Duration) {
	// Drain any stale value to prevent blocking when the loop isn't running.
	select {
	case <-m.changeIntervalChan:
	default:
	}
	select {
	case m.changeIntervalChan <- d:
	case <-m.done:
	}
}

func (m *measurementsSender) SetExtraAttributes(extraAttributes map[string]string) {
	// Drain any stale value to prevent blocking when the loop isn't running.
	select {
	case <-m.changeAttributesChan:
	default:
	}
	select {
	case m.changeAttributesChan <- extraAttributes:
	case <-m.done:
	}
}

func (m *measurementsSender) Stop() {
	m.mux.Lock()
	defer m.mux.Unlock()

	if !m.isRunning {
		return
	}

	close(m.done)
	m.wg.Wait()

	m.isRunning = false
}

func (m *measurementsSender) loop() {
	t := newTicker()
	t.SetInterval(m.interval)
	defer t.Stop()

	for {
		select {
		case newInterval := <-m.changeIntervalChan:
			m.interval = newInterval
			t.SetInterval(newInterval)
		case newAttributes := <-m.changeAttributesChan:
			m.extraAttributes = newAttributes
		case <-m.done:
			return
		case <-t.Chan():
			if m.reporter == nil {
				// Continue if no reporter available
				m.logger.Debug("No reporter, skipping sending measurements.")
				continue
			}

			metrics := m.reporter.OTLPMeasurements(m.extraAttributes)
			if metrics.DataPointCount() == 0 {
				// don't report empty payloads
				continue
			}

			// Send metrics as snappy-encoded otlp proto
			marshaller := pmetric.ProtoMarshaler{}
			marshalled, err := marshaller.MarshalMetrics(metrics)
			if err != nil {
				m.logger.Error("Failed to marshal throughput metrics.", zap.Error(err))
				continue
			}

			encoded := snappy.Encode(nil, marshalled)

			cm := &protobufs.CustomMessage{
				Capability: measurements.ReportMeasurementsV1Capability,
				Type:       measurements.ReportMeasurementsType,
				Data:       encoded,
			}

			success := false
			for i := 0; i < maxSendRetries; i++ {
				sendingChannel, err := m.opampClient.SendCustomMessage(cm)
				switch {
				case err == nil: // OK
					success = true
					m.lastReportedSequence.Add(1)
				case errors.Is(err, types.ErrCustomMessagePending):
					if i == maxSendRetries-1 {
						// Bail out early, since we aren't going to try to send again
						m.logger.Warn("Measurements were blocked by other custom messages, skipping...", zap.Int("retries", maxSendRetries))
						break
					}

					select {
					case <-sendingChannel:
						continue
					case <-m.done:
						return
					}
				default:
					m.logger.Error("Failed to report measurements", zap.Error(err))
				}
				break
			}

			if !success {
				m.logger.Warn("Failed to send measurements after all retries")
			}
		}
	}
}

// ticker is essentially time.ticker, but it provides a SetInterval method
// that allows the interval to be changed. It also allows the interval
// to be configured to a negative or zero duration, in which case the ticker
// never fires.
type ticker struct {
	duration time.Duration
	ticker   *time.Ticker
}

func newTicker() *ticker {
	return &ticker{}
}

func (t *ticker) SetInterval(d time.Duration) {
	if t.duration == d {
		// Nothing to do, this is already the interval
		return
	}

	t.duration = d

	if t.ticker != nil {
		t.ticker.Stop()
		t.ticker = nil
	}

	if d <= 0 {
		// Cannot make a ticker with zero or negative duration;
		// Attempts to use the channel will give a permanently blocking channel.
		return
	}

	t.ticker = time.NewTicker(d)
}

func (t *ticker) Chan() <-chan time.Time {
	if t.ticker == nil {
		// ticker never triggers if 0 or negative duration
		return make(<-chan time.Time)
	}
	return t.ticker.C
}

func (t *ticker) Stop() {
	if t.ticker != nil {
		t.ticker.Stop()
		t.ticker = nil
	}
}
