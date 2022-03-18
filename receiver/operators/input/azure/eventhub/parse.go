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

package eventhub

import (
	"context"

	azhub "github.com/Azure/azure-event-hubs-go/v3"
	"github.com/observiq/observiq-otel-collector/receiver/operators/input/azure"
	"go.uber.org/zap"
)

// handleEvent handles an event received by an Event Hub consumer.
func (e *Input) handleEvent(ctx context.Context, event *azhub.Event) error {
	e.WG.Add(1)
	defer e.WG.Done()

	entry, err := e.NewEntry(nil)
	if err != nil {
		e.Errorw("", zap.Error(err))
		return err
	}

	if err := azure.ParseEvent(*event, entry); err != nil {
		e.Errorw("", zap.Error(err))
		return err
	}

	e.Write(ctx, entry)
	return nil
}
