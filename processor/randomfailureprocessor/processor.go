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

package randomfailureprocessor

import (
	"context"
	"errors"
	"math/rand/v2"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

type randomFailureProcessor struct {
	cfg                   *Config
	randomChanceGenerator func() float64
}

func newRandomFailureProcessor(cfg *Config) *randomFailureProcessor {
	return &randomFailureProcessor{
		cfg:                   cfg,
		randomChanceGenerator: rand.Float64,
	}
}

func (p *randomFailureProcessor) start(_ context.Context, host component.Host) error {
	return nil
}

func (p *randomFailureProcessor) stop(ctx context.Context) error {
	return nil
}

func (p *randomFailureProcessor) processTraces(_ context.Context, td ptrace.Traces) (ptrace.Traces, error) {
	if p.randomChanceGenerator() < p.cfg.FailureRate {
		return td, errors.New("random failure")
	}
	return td, nil
}

func (p *randomFailureProcessor) processLogs(_ context.Context, ld plog.Logs) (plog.Logs, error) {
	if p.randomChanceGenerator() < p.cfg.FailureRate {
		return ld, errors.New("random failure")
	}
	return ld, nil
}

func (p *randomFailureProcessor) processMetrics(_ context.Context, md pmetric.Metrics) (pmetric.Metrics, error) {
	if p.randomChanceGenerator() < p.cfg.FailureRate {
		return md, errors.New("random failure")
	}
	return md, nil
}
