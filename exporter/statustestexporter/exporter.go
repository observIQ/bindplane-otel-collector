// Copyright  observIQ, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package statustestexporter

import (
	"context"
	"errors"
	"math/rand/v2"
	"sync"
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componentstatus"
	"go.uber.org/zap"
)

// statusTestExporter starts successfully and then periodically reports a randomly
// chosen component status (OK, RecoverableError, or PermanentError) to exercise
// the status-reporting pipeline. All telemetry received is silently discarded.
type statusTestExporter struct {
	cfg    *Config
	host   component.Host
	stopCh chan struct{}
	wg     sync.WaitGroup
	logger *zap.Logger
}

func newStatusTestExporter(cfg *Config, logger *zap.Logger) *statusTestExporter {
	return &statusTestExporter{
		cfg:    cfg,
		logger: logger,
	}
}

// Start begins the background status-cycling goroutine.
func (e *statusTestExporter) Start(_ context.Context, host component.Host) error {
	e.host = host
	e.stopCh = make(chan struct{})
	e.wg.Add(1)
	go e.statusLoop()
	return nil
}

// Shutdown stops the background goroutine and waits for it to exit.
func (e *statusTestExporter) Shutdown(_ context.Context) error {
	close(e.stopCh)
	e.wg.Wait()
	return nil
}

func (e *statusTestExporter) statusLoop() {
	defer e.wg.Done()

	ticker := time.NewTicker(e.cfg.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			e.reportRandomStatus()
		case <-e.stopCh:
			return
		}
	}
}

// reportRandomStatus picks a status at random and reports it via componentstatus.
func (e *statusTestExporter) reportRandomStatus() {
	statuses := []componentstatus.Status{
		componentstatus.StatusOK,
		componentstatus.StatusRecoverableError,
	}

	status := statuses[rand.IntN(len(statuses))]

	var event *componentstatus.Event
	switch status {
	case componentstatus.StatusRecoverableError:
		event = componentstatus.NewRecoverableErrorEvent(errors.New("simulated recoverable error"))
	case componentstatus.StatusPermanentError:
		event = componentstatus.NewPermanentErrorEvent(errors.New("simulated permanent error"))
	default:
		event = componentstatus.NewEvent(componentstatus.StatusOK)
	}

	e.logger.Info("Reporting status", zap.String("status", status.String()))
	componentstatus.ReportStatus(e.host, event)
}
