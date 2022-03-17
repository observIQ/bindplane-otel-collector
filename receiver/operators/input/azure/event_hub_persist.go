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

package azure

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/Azure/azure-event-hubs-go/v3/persist"
	"github.com/open-telemetry/opentelemetry-log-collection/operator"
)

// Persister implements persist.CheckpointPersister
type Persister struct {
	DB operator.Persister
}

// Write records an Azure Event Hub Checkpoint to the Stanza persistence backend
func (p *Persister) Write(namespace, name, consumerGroup, partitionID string, checkpoint persist.Checkpoint) error {
	key := p.persistenceKey(namespace, name, consumerGroup, partitionID)
	value, err := json.Marshal(checkpoint)
	if err != nil {
		return err
	}
	return p.DB.Set(context.TODO(), key, value)
}

// Read retrieves an Azure Event Hub Checkpoint from the Stanza persistence backend
func (p *Persister) Read(namespace, name, consumerGroup, partitionID string) (persist.Checkpoint, error) {
	key := p.persistenceKey(namespace, name, consumerGroup, partitionID)
	value, err := p.DB.Get(context.TODO(), key)
	if err != nil {
		return persist.Checkpoint{}, err
	}

	if len(value) < 1 {
		return persist.Checkpoint{}, nil
	}

	var checkpoint persist.Checkpoint
	err = json.Unmarshal(value, &checkpoint)
	return checkpoint, err
}

func (p *Persister) persistenceKey(namespace, name, consumerGroup, partitionID string) string {
	x := fmt.Sprintf("%s-%s-%s-%s", namespace, name, consumerGroup, partitionID)
	return x
}
