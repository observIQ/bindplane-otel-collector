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

package statustestprocessor

import (
	"context"
	"errors"
	"math/rand/v2"
	"sync"
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componentstatus"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.uber.org/zap"
)

// statusTestProcessor starts successfully and then periodically reports a randomly
// chosen component status (OK, RecoverableError, or PermanentError) to exercise
// the status-reporting pipeline. All telemetry is passed through unchanged.
type statusTestProcessor struct {
	cfg    *Config
	host   component.Host
	stopCh chan struct{}
	wg     sync.WaitGroup
	logger *zap.Logger
}

func newStatusTestProcessor(cfg *Config, logger *zap.Logger) *statusTestProcessor {
	return &statusTestProcessor{
		cfg:    cfg,
		logger: logger,
	}
}

func (p *statusTestProcessor) start(_ context.Context, host component.Host) error {
	p.host = host
	p.stopCh = make(chan struct{})
	p.wg.Add(1)
	go p.statusLoop()
	return nil
}

func (p *statusTestProcessor) stop(_ context.Context) error {
	close(p.stopCh)
	p.wg.Wait()
	return nil
}

func (p *statusTestProcessor) statusLoop() {
	defer p.wg.Done()

	ticker := time.NewTicker(p.cfg.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			p.reportRandomStatus()
		case <-p.stopCh:
			return
		}
	}
}

// reportRandomStatus picks a status at random and reports it via componentstatus.
func (p *statusTestProcessor) reportRandomStatus() {
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

	p.logger.Info("Reporting status", zap.String("status", status.String()))
	componentstatus.ReportStatus(p.host, event)
}

// processLogs passes logs through unchanged.
func (p *statusTestProcessor) processLogs(_ context.Context, ld plog.Logs) (plog.Logs, error) {
	return ld, nil
}

// processMetrics passes metrics through unchanged.
func (p *statusTestProcessor) processMetrics(_ context.Context, md pmetric.Metrics) (pmetric.Metrics, error) {
	return md, nil
}

// processTraces passes traces through unchanged.
func (p *statusTestProcessor) processTraces(_ context.Context, td ptrace.Traces) (ptrace.Traces, error) {
	return td, nil
}
