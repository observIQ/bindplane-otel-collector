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

package topologyprocessor

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"go.opentelemetry.io/collector/client"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.uber.org/zap"
)

const (
	organizationIDHeader = "X-Bindplane-Organization-ID"
	accountIDHeader      = "X-Bindplane-Account-ID"
	configurationHeader  = "X-Bindplane-Configuration"
	resourceNameHeader   = "X-Bindplane-Resource-Name"
)

type topologyProcessor struct {
	logger      *zap.Logger
	topology    *TopoState
	processorID component.ID
	bindplane   *component.ID

	startOnce sync.Once
}

// newTopologyProcessor creates a new topology processor
func newTopologyProcessor(logger *zap.Logger, cfg *Config, processorID component.ID) (*topologyProcessor, error) {
	destGw := GatewayInfo{
		GatewayID:      strings.TrimPrefix(processorID.String(), "topology/"),
		Configuration:  cfg.Configuration,
		AccountID:      cfg.AccountID,
		OrganizationID: cfg.OrganizationID,
	}
	topology, err := NewTopologyState(destGw)
	if err != nil {
		return nil, fmt.Errorf("create topology state: %w", err)
	}

	return &topologyProcessor{
		logger:      logger,
		topology:    topology,
		processorID: processorID,
		bindplane:   cfg.BindplaneExtension,
		startOnce:   sync.Once{},
	}, nil
}

func (tp *topologyProcessor) start(_ context.Context, host component.Host) error {
	var err error
	tp.startOnce.Do(func() {
		registry, getRegErr := GetTopologyRegistry(host, tp.bindplane)
		if getRegErr != nil {
			err = fmt.Errorf("get topology registry: %w", getRegErr)
			return
		}

		if registry != nil {
			registerErr := registry.RegisterTopologyState(tp.processorID.String(), tp.topology)
			if registerErr != nil {
				return
			}
		}
	})

	return err
}

func (tp *topologyProcessor) processTraces(ctx context.Context, td ptrace.Traces) (ptrace.Traces, error) {
	tp.processTopologyHeaders(ctx)
	return td, nil
}

func (tp *topologyProcessor) processLogs(ctx context.Context, ld plog.Logs) (plog.Logs, error) {
	tp.processTopologyHeaders(ctx)
	return ld, nil
}

func (tp *topologyProcessor) processMetrics(ctx context.Context, md pmetric.Metrics) (pmetric.Metrics, error) {
	tp.processTopologyHeaders(ctx)
	return md, nil
}

func (tp *topologyProcessor) processTopologyHeaders(ctx context.Context) {
	headers := client.FromContext(ctx).Metadata
	var configuration, accountID, organizationID, resourceName string

	configurationHeaders := headers.Get(configurationHeader)
	if len(configurationHeaders) > 0 {
		configuration = configurationHeaders[0]
	} else {
		return
	}

	accountIDHeaders := headers.Get(accountIDHeader)
	if len(accountIDHeaders) > 0 {
		accountID = accountIDHeaders[0]
	} else {
		return
	}

	organizationIDHeaders := headers.Get(organizationIDHeader)
	if len(organizationIDHeaders) > 0 {
		organizationID = organizationIDHeaders[0]
	} else {
		return
	}

	resourceNameHeaders := headers.Get(resourceNameHeader)
	if len(resourceNameHeaders) > 0 {
		resourceName = resourceNameHeaders[0]
	} else {
		return
	}

	// only upsert if all headers are present
	if configuration != "" && accountID != "" && organizationID != "" && resourceName != "" {
		gw := GatewayInfo{
			Configuration:  configuration,
			AccountID:      accountID,
			OrganizationID: organizationID,
			GatewayID:      resourceName,
		}
		tp.topology.UpsertRoute(ctx, gw)
	}
}

func (tp *topologyProcessor) shutdown(_ context.Context) error {
	unregisterProcessor(tp.processorID)
	return nil
}
