// Copyright observIQ, Inc.
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

package pcapreceiver

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.uber.org/zap"
)

// pcapReceiver captures network packets and converts them to logs
type pcapReceiver struct {
	config   *Config
	logger   *zap.Logger
	consumer consumer.Logs
	cancel   context.CancelFunc
}

func newReceiver(
	config *Config,
	logger *zap.Logger,
	consumer consumer.Logs,
) *pcapReceiver {
	return &pcapReceiver{
		config:   config,
		logger:   logger,
		consumer: consumer,
	}
}

func (r *pcapReceiver) Start(ctx context.Context, _ component.Host) error {
	// To be implemented in Phase 7
	return nil
}

func (r *pcapReceiver) Shutdown(_ context.Context) error {
	// To be implemented in Phase 7
	return nil
}

