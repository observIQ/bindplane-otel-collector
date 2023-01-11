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

package sapnetweaverreceiver // import "github.com/observiq/observiq-otel-collector/receiver/sapnetweaverreceiver"

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/hooklift/gowsdl/soap"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/receiver"
	"go.opentelemetry.io/collector/receiver/scrapererror"

	"github.com/observiq/observiq-otel-collector/receiver/sapnetweaverreceiver/internal/metadata"
	"github.com/observiq/observiq-otel-collector/receiver/sapnetweaverreceiver/internal/models"
)

type sapNetweaverScraper struct {
	settings component.TelemetrySettings
	cfg      *Config
	client   *soap.Client
	service  webService
	instance string
	hostname string
	mb       *metadata.MetricsBuilder
}

func newSapNetweaverScraper(
	settings receiver.CreateSettings,
	cfg *Config,
) *sapNetweaverScraper {
	a := &sapNetweaverScraper{
		settings: settings.TelemetrySettings,
		cfg:      cfg,
		mb:       metadata.NewMetricsBuilder(cfg.Metrics, settings.BuildInfo),
	}

	return a
}

func (s *sapNetweaverScraper) start(_ context.Context, host component.Host) error {
	soapClient, err := newSoapClient(s.cfg, host, s.settings)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}
	s.client = soapClient
	s.service = newWebService(s.client)

	return nil
}

func (s *sapNetweaverScraper) GetCurrentInstance() error {
	var response *models.GetInstancePropertiesResponse
	response, err := s.service.GetInstanceProperties()
	if err != nil {
		return err
	}

	for _, prop := range response.Properties {
		switch prop.Property {
		case "INSTANCE_NAME":
			s.instance = prop.Value
		case "SAPLOCALHOST":
			s.hostname = prop.Value
		}
	}

	return nil
}

func (s *sapNetweaverScraper) scrape(ctx context.Context) (pmetric.Metrics, error) {
	if s.client == nil || s.service == nil {
		return pmetric.Metrics{}, errors.New("failed to create client")
	}

	errs := &scrapererror.ScrapeErrors{}
	err := s.GetCurrentInstance()
	if err != nil {
		errs.AddPartial(1, fmt.Errorf("failed to get current instance details: %w", err))
	}

	s.collectMetrics(ctx, errs)
	return s.mb.Emit(), errs.Combine()
}

func (s *sapNetweaverScraper) collectMetrics(ctx context.Context, errs *scrapererror.ScrapeErrors) {
	now := pcommon.NewTimestampFromTime(time.Now())
	s.collectAlertTree(ctx, now, errs)
	s.collectEnqGetLockTable(ctx, now, errs)
	s.mb.EmitForResource(metadata.WithSapnetweaverInstance(s.instance), metadata.WithSapnetweaverNode(s.hostname))
}

func (s *sapNetweaverScraper) collectAlertTree(_ context.Context, now pcommon.Timestamp, errs *scrapererror.ScrapeErrors) {
	alertTreeResponse := map[string]string{}
	alertTree, err := s.service.GetAlertTree()
	if err != nil {
		errs.AddPartial(1, fmt.Errorf("failed to collect Alert Tree metrics: %w", err))
		return
	}

	for _, node := range alertTree.AlertNode {
		value := strings.Split(node.Description, " ")
		alertTreeResponse[node.Name] = value[0]
		if node.Name == "ICM" {
			alertTreeResponse[node.Name] = string(node.ActualValue)
		}
	}

	s.recordSapnetweaverHostCPUUtilizationDataPoint(now, alertTreeResponse, errs)
	s.recordSapnetweaverHostMemoryVirtualOverheadDataPoint(now, alertTreeResponse, errs)
	s.recordSapnetweaverHostMemoryVirtualSwapDataPoint(now, alertTreeResponse, errs)
	s.recordSapnetweaverSessionsHTTPCountDataPoint(now, alertTreeResponse, errs)
	s.recordCurrentSecuritySessions(now, alertTreeResponse, errs)
	s.recordSapnetweaverWorkProcessesActiveCount(now, alertTreeResponse, errs)
	s.recordSapnetweaverSessionsWebCountDataPoint(now, alertTreeResponse, errs)
	s.recordSapnetweaverSessionsBrowserCountDataPoint(now, alertTreeResponse, errs)
	s.recordSapnetweaverSessionsEjbCountDataPoint(now, alertTreeResponse, errs)
	s.recordSapnetweaverIcmAvailabilityDataPoint(now, alertTreeResponse, errs)
	s.recordSapnetweaverHostSpoolListUsedDataPoint(now, alertTreeResponse, errs)
	s.recordSapnetweaverShortDumpsCountDataPoint(now, alertTreeResponse, errs)
}

func (s *sapNetweaverScraper) collectEnqGetLockTable(_ context.Context, now pcommon.Timestamp, errs *scrapererror.ScrapeErrors) {
	lockTable, err := s.service.EnqGetLockTable()
	if err != nil {
		errs.AddPartial(1, fmt.Errorf("failed to collect Enq Lock Table metrics: %w", err))
		return
	}

	s.mb.RecordSapnetweaverLocksEnqueueCountDataPoint(now, int64(len(lockTable.EnqLock)))
}
