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

package sapnetweaverreceiver // import "github.com/observiq/bindplane-otel-collector/receiver/sapnetweaverreceiver"

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
	"go.opentelemetry.io/collector/scraper/scrapererror"

	"github.com/observiq/bindplane-otel-collector/receiver/sapnetweaverreceiver/internal/metadata"
	"github.com/observiq/bindplane-otel-collector/receiver/sapnetweaverreceiver/internal/models"
)

type sapNetweaverScraper struct {
	settings component.TelemetrySettings
	cfg      *Config
	client   *soap.Client
	service  webService
	instance string
	hostname string
	SID      string
	rb       *metadata.ResourceBuilder
	mb       *metadata.MetricsBuilder
}

func newSapNetweaverScraper(
	settings receiver.Settings,
	cfg *Config,
) *sapNetweaverScraper {
	a := &sapNetweaverScraper{
		settings: settings.TelemetrySettings,
		cfg:      cfg,
		rb:       metadata.NewResourceBuilder(cfg.MetricsBuilderConfig.ResourceAttributes),
		mb:       metadata.NewMetricsBuilder(cfg.MetricsBuilderConfig, settings),
	}

	return a
}

func (s *sapNetweaverScraper) start(ctx context.Context, host component.Host) error {
	soapClient, err := newSoapClient(ctx, s.cfg, host, s.settings)
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
		case "SAPSYSTEMNAME":
			s.SID = prop.Value
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
		errs.AddPartial(1, fmt.Errorf("failed to collect GetInstanceProperties metrics: %w", err))
	}

	s.collectMetrics(ctx, errs)
	return s.mb.Emit(), errs.Combine()
}

func (s *sapNetweaverScraper) collectMetrics(ctx context.Context, errs *scrapererror.ScrapeErrors) {
	now := pcommon.NewTimestampFromTime(time.Now())
	s.collectGetAlertTree(ctx, now, errs)
	s.collectABAPGetSystemWPTable(ctx, now, errs)
	s.collectEnqGetStatistic(ctx, now, errs)
	s.collectGetQueueStatistic(ctx, now, errs)
	s.collectGetProcessList(ctx, now, errs)
	s.collectGetSystemInstanceList(ctx, now, errs)

	s.rb.SetSapnetweaverInstance(s.instance)
	s.rb.SetSapnetweaverNode(s.hostname)
	s.rb.SetSapnetweaverSID(s.SID)
	s.mb.EmitForResource(metadata.WithResource(s.rb.Emit()))
}

// collectABAPGetSystemWPTable collects metrics from the ABAPGetSystemWPTable method
func (s *sapNetweaverScraper) collectABAPGetSystemWPTable(_ context.Context, now pcommon.Timestamp, errs *scrapererror.ScrapeErrors) {
	abapGetSystemWPTable, err := s.service.ABAPGetSystemWPTable()
	if err != nil {
		errs.AddPartial(1, fmt.Errorf("failed to collect ABAPGetSystemWPTable metrics: %w", err))
		return
	}

	s.recordSapnetweaverWorkProcessActiveCountDataPoint(now, abapGetSystemWPTable, errs)
}

// collectEnqGetStatistic collects metrics from the EnqGetStatistic method
func (s *sapNetweaverScraper) collectEnqGetStatistic(_ context.Context, now pcommon.Timestamp, errs *scrapererror.ScrapeErrors) {
	enqGetStatistic, err := s.service.EnqGetStatistic()
	if err != nil {
		errs.AddPartial(1, fmt.Errorf("failed to collect EnqGetStatistic metrics: %w", err))
		return
	}

	s.recordSapnetweaverLocksDataPoints(now, enqGetStatistic, errs)
}

// collectGetQueueStatistic collects metrics from the GetQueueStatistic method
func (s *sapNetweaverScraper) collectGetQueueStatistic(_ context.Context, now pcommon.Timestamp, errs *scrapererror.ScrapeErrors) {
	getQueueStatistic, err := s.service.GetQueueStatistic()
	if err != nil {
		errs.AddPartial(1, fmt.Errorf("failed to collect GetQueueStatistic metrics: %w", err))
		return
	}

	s.recordSapnetweaverQueueDataPoints(now, getQueueStatistic, errs)
}

// collectGetProcessList collects metrics from the GetProcessList method
func (s *sapNetweaverScraper) collectGetProcessList(_ context.Context, now pcommon.Timestamp, errs *scrapererror.ScrapeErrors) {
	getProcessList, err := s.service.GetProcessList()
	if err != nil {
		errs.AddPartial(1, fmt.Errorf("failed to collect GetProcessList metrics: %w", err))
		return
	}

	s.recordSapnetweaverProcessAvailabilityDataPoint(now, getProcessList, errs)
}

// collectGetSystemInstanceList collects metrics from the GetSystemInstanceList method
func (s *sapNetweaverScraper) collectGetSystemInstanceList(_ context.Context, now pcommon.Timestamp, errs *scrapererror.ScrapeErrors) {
	getSystemInstanceList, err := s.service.GetSystemInstanceList()
	if err != nil {
		errs.AddPartial(1, fmt.Errorf("failed to collect GetSystemInstanceList metrics: %w", err))
		return
	}

	s.recordSapnetweaverSystemInstanceAvailabilityDataPoint(now, getSystemInstanceList, errs)
}

// collectGetAlertTree collects metrics from the GetAlertTree method
func (s *sapNetweaverScraper) collectGetAlertTree(_ context.Context, now pcommon.Timestamp, errs *scrapererror.ScrapeErrors) {
	alertTreeResponse := map[string]string{}
	alertTree, err := s.service.GetAlertTree()
	if err != nil {
		errs.AddPartial(1, fmt.Errorf("failed to collect Alert Tree metrics: %w", err))
		return
	}

	toggleSwapSpaceFlag := false
	for _, node := range alertTree.AlertNode {
		value := strings.Split(node.Description, " ")
		alertTreeResponse[node.Name] = value[0]
		if node.Name == "ICM" || node.Name == "AbapErrorInUpdate" || node.Name == "AbortedJobs" {
			alertTreeResponse[node.Name] = string(node.ActualValue)
		}

		// There are multiple "Percentage_Used" fields with no unique column identifiers.
		// The wanted "Percentage_Used" comes ~2 rows after the Swap_Space.
		if node.Name == "Swap_Space" {
			toggleSwapSpaceFlag = true
		}
		if toggleSwapSpaceFlag && node.Name == "Percentage_Used" {
			alertTreeResponse["Swap_Space_Percentage_Used"] = value[0]
			toggleSwapSpaceFlag = false
		}
	}

	s.recordSapnetweaverDatabaseDialogRequestTimeDataPoint(now, alertTreeResponse, errs)
	s.recordSapnetweaverCPUUtilizationDataPoint(now, alertTreeResponse, errs)
	s.recordSapnetweaverCPUSystemUtilizationDataPoint(now, alertTreeResponse, errs)
	s.recordSapnetweaverSpoolRequestErrorCountDataPoint(now, alertTreeResponse, errs)
	s.recordSapnetweaverWorkProcessJobAbortedCountDataPoint(now, alertTreeResponse, errs)

	s.recordSapnetweaverMemorySwapSpaceUtilizationDataPoint(now, alertTreeResponse, errs)
	s.recordSapnetweaverMemoryConfiguredDataPoint(now, alertTreeResponse, errs)
	s.recordSapnetweaverMemoryFreeDataPoint(now, alertTreeResponse, errs)
	s.recordSapnetweaverSessionCountDataPoint(now, alertTreeResponse, errs)
	s.recordSapnetweaverAbapUpdateStatusDataPoint(now, alertTreeResponse, errs)
	s.recordSapnetweaverResponseDurationDataPoint(now, alertTreeResponse, errs)
	s.recordSapnetweaverRequestCountDataPoint(now, alertTreeResponse, errs)
	s.recordSapnetweaverRequestTimeoutCountDataPoint(now, alertTreeResponse, errs)
	s.recordSapnetweaverConnectionErrorCountDataPoint(now, alertTreeResponse, errs)
	s.recordSapnetweaverCacheEvictionsDataPoint(now, alertTreeResponse, errs)
	s.recordSapnetweaverCacheHitsDataPoint(now, alertTreeResponse, errs)

	s.recordSapnetweaverHostSpoolListUtilizationDataPoint(now, alertTreeResponse, errs)
	s.recordSapnetweaverShortDumpsCountDataPoint(now, alertTreeResponse, errs)
	s.recordSapnetweaverHostMemoryVirtualOverheadDataPoint(now, alertTreeResponse, errs)
	s.recordSapnetweaverHostMemoryVirtualSwapDataPoint(now, alertTreeResponse, errs)
	s.recordSapnetweaverSessionsHTTPCountDataPoint(now, alertTreeResponse, errs)
	s.recordCurrentSecuritySessions(now, alertTreeResponse, errs)
	s.recordSapnetweaverSessionsWebCountDataPoint(now, alertTreeResponse, errs)
	s.recordSapnetweaverSessionsBrowserCountDataPoint(now, alertTreeResponse, errs)
	s.recordSapnetweaverSessionsEjbCountDataPoint(now, alertTreeResponse, errs)
}
