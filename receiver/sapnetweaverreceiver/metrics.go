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
	"errors"
	"fmt"
	"strconv"
	"strings"

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
	errValueHyphen       = errors.New("'-' value found")
	errInvalidStateColor = errors.New("invalid control state color value")
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

	switch models.StateColor(val) {
	case models.StateColorGray:
		s.mb.RecordSapnetweaverIcmAvailabilityDataPoint(now, 1, metadata.AttributeControlStateGrey)
		s.mb.RecordSapnetweaverIcmAvailabilityDataPoint(now, 0, metadata.AttributeControlStateGreen)
		s.mb.RecordSapnetweaverIcmAvailabilityDataPoint(now, 0, metadata.AttributeControlStateYellow)
		s.mb.RecordSapnetweaverIcmAvailabilityDataPoint(now, 0, metadata.AttributeControlStateRed)
	case models.StateColorGreen:
		s.mb.RecordSapnetweaverIcmAvailabilityDataPoint(now, 0, metadata.AttributeControlStateGrey)
		s.mb.RecordSapnetweaverIcmAvailabilityDataPoint(now, 1, metadata.AttributeControlStateGreen)
		s.mb.RecordSapnetweaverIcmAvailabilityDataPoint(now, 0, metadata.AttributeControlStateYellow)
		s.mb.RecordSapnetweaverIcmAvailabilityDataPoint(now, 0, metadata.AttributeControlStateRed)
	case models.StateColorYellow:
		s.mb.RecordSapnetweaverIcmAvailabilityDataPoint(now, 0, metadata.AttributeControlStateGrey)
		s.mb.RecordSapnetweaverIcmAvailabilityDataPoint(now, 0, metadata.AttributeControlStateGreen)
		s.mb.RecordSapnetweaverIcmAvailabilityDataPoint(now, 1, metadata.AttributeControlStateYellow)
		s.mb.RecordSapnetweaverIcmAvailabilityDataPoint(now, 0, metadata.AttributeControlStateRed)
	case models.StateColorRed:
		s.mb.RecordSapnetweaverIcmAvailabilityDataPoint(now, 0, metadata.AttributeControlStateGrey)
		s.mb.RecordSapnetweaverIcmAvailabilityDataPoint(now, 0, metadata.AttributeControlStateGreen)
		s.mb.RecordSapnetweaverIcmAvailabilityDataPoint(now, 0, metadata.AttributeControlStateYellow)
		s.mb.RecordSapnetweaverIcmAvailabilityDataPoint(now, 1, metadata.AttributeControlStateRed)
	default:
		errs.AddPartial(1, fmt.Errorf(collectMetricError, metricName, errInvalidStateColor))
	}
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

	if strings.Contains(val, "-") {
		return "", fmt.Errorf(collectMetricError, metricName, errValueHyphen)
	}
	return val, nil
}
