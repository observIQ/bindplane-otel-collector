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
	"fmt"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/receiver/scrapererror"

	"github.com/observiq/observiq-otel-collector/receiver/sapnetweaverreceiver/internal/metadata"
	"github.com/observiq/observiq-otel-collector/receiver/sapnetweaverreceiver/internal/models"
)

const (
	collectMetricError = "failed to collect metric %s: %w"
	// MBToBytes converts 1 megabytes to byte
	MBToBytes = 1000000
)

var (
	errValueNotFound     = errors.New("value not found")
	errValueEmpty        = errors.New("value Empty _")
	errInvalidStateColor = errors.New("invalid STATECOLOR value")
)

func (s *sapNetweaverScraper) recordSapnetweaverHostMemoryVirtualOverheadDataPoint(now pcommon.Timestamp, alertTreeResponse map[string]string, errs *scrapererror.ScrapeErrors) {
	metricName := "Memory Overhead"
	val, err := parseResponse(metricName, alertTreeResponse)
	if err != nil {
		errs.AddPartial(1, err)
		return
	}

	mbytes, err := strconv.ParseInt(val, 10, 64)
	if err != nil {
		errs.AddPartial(1, fmt.Errorf("failed to parse int64 for SapnetweaverHostMemoryVirtualOverhead, value was %v: %w", val, err))
		return
	}

	s.mb.RecordSapnetweaverHostMemoryVirtualOverheadDataPoint(now, mbytes*int64(MBToBytes))
}

func (s *sapNetweaverScraper) recordSapnetweaverHostMemoryVirtualSwapDataPoint(now pcommon.Timestamp, alertTreeResponse map[string]string, errs *scrapererror.ScrapeErrors) {
	metricName := "Memory Swapped Out"
	val, err := parseResponse(metricName, alertTreeResponse)
	if err != nil {
		errs.AddPartial(1, err)
		return
	}

	MBToBytes := 1000000
	mbytes, err := strconv.ParseInt(val, 10, 64)
	if err != nil {
		errs.AddPartial(1, fmt.Errorf("failed to parse int64 for SapnetweaverHostMemoryVirtualSwap, value was %v: %w", val, err))
		return
	}

	s.mb.RecordSapnetweaverHostMemoryVirtualSwapDataPoint(now, mbytes*int64(MBToBytes))
}

func (s *sapNetweaverScraper) recordSapnetweaverSessionsHTTPCountDataPoint(now pcommon.Timestamp, alertTreeResponse map[string]string, errs *scrapererror.ScrapeErrors) {
	metricName := "CurrentHttpSessions"
	val, err := parseResponse(metricName, alertTreeResponse)
	if err != nil {
		errs.AddPartial(1, err)
		return
	}

	err = s.mb.RecordSapnetweaverSessionsHTTPCountDataPoint(now, val)
	if err != nil {
		errs.AddPartial(1, err)
		return
	}
}

func (s *sapNetweaverScraper) recordCurrentSecuritySessions(now pcommon.Timestamp, alertTreeResponse map[string]string, errs *scrapererror.ScrapeErrors) {
	metricName := "CurrentSecuritySessions"
	val, err := parseResponse(metricName, alertTreeResponse)
	if err != nil {
		errs.AddPartial(1, err)
		return
	}

	err = s.mb.RecordSapnetweaverSessionsSecurityCountDataPoint(now, val)
	if err != nil {
		errs.AddPartial(1, err)
		return
	}
}

func (s *sapNetweaverScraper) recordSapnetweaverWorkProcessesActiveCount(now pcommon.Timestamp, alertTreeResponse map[string]string, errs *scrapererror.ScrapeErrors) {
	metricName := "Total Number of Work Processes"
	val, err := parseResponse(metricName, alertTreeResponse)
	if err != nil {
		errs.AddPartial(1, err)
		return
	}

	err = s.mb.RecordSapnetweaverWorkProcessesActiveCountDataPoint(now, val)
	if err != nil {
		errs.AddPartial(1, err)
		return
	}
}

func (s *sapNetweaverScraper) recordSapnetweaverSessionsWebCountDataPoint(now pcommon.Timestamp, alertTreeResponse map[string]string, errs *scrapererror.ScrapeErrors) {
	metricName := "Web Sessions"
	val, err := parseResponse(metricName, alertTreeResponse)
	if err != nil {
		errs.AddPartial(1, err)
		return
	}

	err = s.mb.RecordSapnetweaverSessionsWebCountDataPoint(now, val)
	if err != nil {
		errs.AddPartial(1, err)
		return
	}
}

func (s *sapNetweaverScraper) recordSapnetweaverSessionsBrowserCountDataPoint(now pcommon.Timestamp, alertTreeResponse map[string]string, errs *scrapererror.ScrapeErrors) {
	metricName := "Browser Sessions"
	val, err := parseResponse(metricName, alertTreeResponse)
	if err != nil {
		errs.AddPartial(1, err)
		return
	}

	err = s.mb.RecordSapnetweaverSessionsBrowserCountDataPoint(now, val)
	if err != nil {
		errs.AddPartial(1, err)
		return
	}
}

func (s *sapNetweaverScraper) recordSapnetweaverSessionsEjbCountDataPoint(now pcommon.Timestamp, alertTreeResponse map[string]string, errs *scrapererror.ScrapeErrors) {
	metricName := "EJB Sessions"
	val, err := parseResponse(metricName, alertTreeResponse)
	if err != nil {
		errs.AddPartial(1, err)
		return
	}

	err = s.mb.RecordSapnetweaverSessionsEjbCountDataPoint(now, val)
	if err != nil {
		errs.AddPartial(1, err)
		return
	}
}

func (s *sapNetweaverScraper) recordSapnetweaverIcmAvailabilityDataPoint(now pcommon.Timestamp, alertTreeResponse map[string]string, errs *scrapererror.ScrapeErrors) {
	metricName := "ICM"
	val, ok := alertTreeResponse[metricName]
	if !ok {
		errs.AddPartial(1, fmt.Errorf(collectMetricError, metricName, errValueNotFound))
		return
	}

	stateColorCode, err := stateColorToInt(metricName, models.StateColor(val))
	if err != nil {
		errs.AddPartial(1, fmt.Errorf(collectMetricError, metricName, err))
		return
	}

	stateColorAttribute, err := stateColorToAttribute(metricName, models.StateColor(val))
	if err != nil {
		errs.AddPartial(1, fmt.Errorf(collectMetricError, metricName, err))
		return
	}

	s.mb.RecordSapnetweaverIcmAvailabilityDataPoint(now, stateColorCode, stateColorAttribute)
}

func (s *sapNetweaverScraper) recordSapnetweaverHostCPUUtilizationDataPoint(now pcommon.Timestamp, alertTreeResponse map[string]string, errs *scrapererror.ScrapeErrors) {
	metricName := "CPU_Utilization"
	val, err := parseResponse(metricName, alertTreeResponse)
	if err != nil {
		errs.AddPartial(1, err)
		return
	}

	err = s.mb.RecordSapnetweaverHostCPUUtilizationDataPoint(now, val)
	if err != nil {
		errs.AddPartial(1, err)
		return
	}
}

func (s *sapNetweaverScraper) recordSapnetweaverHostSpoolListUsedDataPoint(now pcommon.Timestamp, alertTreeResponse map[string]string, errs *scrapererror.ScrapeErrors) {
	metricName := "HostspoolListUsed"
	val, err := parseResponse(metricName, alertTreeResponse)
	if err != nil {
		errs.AddPartial(1, err)
		return
	}

	err = s.mb.RecordSapnetweaverHostSpoolListUsedDataPoint(now, val)
	if err != nil {
		errs.AddPartial(1, err)
		return
	}
}

func (s *sapNetweaverScraper) recordSapnetweaverShortDumpsCountDataPoint(now pcommon.Timestamp, alertTreeResponse map[string]string, errs *scrapererror.ScrapeErrors) {
	metricName := "Shortdumps Frequency"
	val, err := parseResponse(metricName, alertTreeResponse)
	if err != nil {
		errs.AddPartial(1, err)
		return
	}

	err = s.mb.RecordSapnetweaverShortDumpsRateDataPoint(now, val)
	if err != nil {
		errs.AddPartial(1, err)
		return
	}
}

func parseResponse(metricName string, alertTreeResponse map[string]string) (string, error) {
	val, ok := alertTreeResponse[metricName]
	if !ok {
		return "", fmt.Errorf(collectMetricError, metricName, errValueNotFound)
	}

	if strings.Contains(val, "_") {
		return "", fmt.Errorf(collectMetricError, metricName, errValueEmpty)
	}
	return val, nil
}

func stateColorToInt(metricName string, statecolor models.StateColor) (int64, error) {
	switch statecolor {
	case models.StateColorGray:
		return int64(models.StateColorCodeGray), nil
	case models.StateColorGreen:
		return int64(models.StateColorCodeGreen), nil
	case models.StateColorYellow:
		return int64(models.StateColorCodeYellow), nil
	case models.StateColorRed:
		return int64(models.StateColorCodeRed), nil
	default:
		return -1, errInvalidStateColor
	}
}

func stateColorToAttribute(metricName string, statecolor models.StateColor) (metadata.AttributeControlState, error) {
	switch statecolor {
	case models.StateColorGray:
		return metadata.AttributeControlStateGrey, nil
	case models.StateColorGreen:
		return metadata.AttributeControlStateGreen, nil
	case models.StateColorYellow:
		return metadata.AttributeControlStateYellow, nil
	case models.StateColorRed:
		return metadata.AttributeControlStateRed, nil
	default:
		return -1, errInvalidStateColor
	}
}
