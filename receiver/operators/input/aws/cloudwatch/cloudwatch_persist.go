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

package cloudwatch

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"

	"github.com/open-telemetry/opentelemetry-log-collection/operator"
)

type persister struct {
	DB operator.Persister
}

// Helper function to get persisted data
func (p *persister) Read(ctx context.Context, key string) (int64, error) {
	startTimeBytes, err := p.DB.Get(ctx, key)
	if err != nil {
		return -1, fmt.Errorf("there was an error reading from persistent storage: %w", err)
	}
	buffer := bytes.NewBuffer(startTimeBytes)
	var startTime int64
	err = binary.Read(buffer, binary.BigEndian, &startTime)
	if err != nil && err.Error() != "EOF" {
		return 0, err
	}
	return startTime, nil
}

// Helper function to set persisted data
func (p *persister) Write(ctx context.Context, key string, value int64) error {
	var buf = make([]byte, 8)
	binary.BigEndian.PutUint64(buf, uint64(value))
	return p.DB.Set(ctx, key, buf)
}
