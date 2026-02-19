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

package statustestreceiver

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

// statusTestReceiver starts successfully and then periodically reports a randomly
// chosen component status (OK, RecoverableError, or PermanentError) to exercise
// the status-reporting pipeline.
type statusTestReceiver struct {
	cfg    *Config
	host   component.Host
	stopCh chan struct{}
	wg     sync.WaitGroup
	logger *zap.Logger
}

func newStatusTestReceiver(cfg *Config, logger *zap.Logger) *statusTestReceiver {
	return &statusTestReceiver{
		cfg:    cfg,
		logger: logger,
	}
}

// Start begins the background status-cycling goroutine.
func (r *statusTestReceiver) Start(_ context.Context, host component.Host) error {
	r.host = host
	r.stopCh = make(chan struct{})
	r.wg.Add(1)
	go r.statusLoop()
	return nil
}

// Shutdown stops the background goroutine and waits for it to exit.
func (r *statusTestReceiver) Shutdown(_ context.Context) error {
	close(r.stopCh)
	r.wg.Wait()
	return nil
}

func (r *statusTestReceiver) statusLoop() {
	defer r.wg.Done()

	ticker := time.NewTicker(r.cfg.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			r.reportRandomStatus()
		case <-r.stopCh:
			return
		}
	}
}

// reportRandomStatus picks a status at random and reports it via componentstatus.
func (r *statusTestReceiver) reportRandomStatus() {
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

	r.logger.Info("Reporting status", zap.String("status", status.String()))
	componentstatus.ReportStatus(r.host, event)
}
