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
	"context"

	"github.com/open-telemetry/opamp-go/client/types"
	"go.uber.org/zap"
)

type zapOpAMPLoggerAdapter struct {
	logger *zap.SugaredLogger
}

var _ types.Logger = (*zapOpAMPLoggerAdapter)(nil)

func newZapOpAMPLoggerAdapter(logger *zap.Logger) *zapOpAMPLoggerAdapter {
	return &zapOpAMPLoggerAdapter{
		logger: logger.Sugar(),
	}
}

func (o zapOpAMPLoggerAdapter) Debugf(_ context.Context, format string, v ...any) {
	o.logger.Debugf(format, v...)
}

func (o zapOpAMPLoggerAdapter) Errorf(_ context.Context, format string, v ...any) {
	o.logger.Errorf(format, v...)
}
