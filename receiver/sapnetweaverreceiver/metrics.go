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
	collectMetricError              = "failed to collect metric %s: %w"
	collectMetricErrorWithAttribute = "failed to collect metric %s with attribute %s: %w"
	// MBToBytes converts 1 megabytes to byte
	MBToBytes = 1000000
)

var (
	errValueNotFound = errors.New("value not found")
	errValueHyphen   = errors.New("'-' value found")
)

func (s *sapNetweaverScraper) recordSapnetweaverSystemInstanceAvailabilityDataPoint(now pcommon.Timestamp, systemInstanceListResponse *models.GetSystemInstanceListResponse, errs *scrapererror.ScrapeErrors) {
	metricName := "Service Availability"
	if systemInstanceListResponse.Instance.Item == nil {
		err := formatErrorMsg(metricName, "", errValueNotFound)
		errs.AddPartial(1, err)
		return
	}

	for _, response := range systemInstanceListResponse.Instance.Item {
		featureList := strings.Split(response.Features, "|")
		for _, feature := range featureList {
			var gray, green, yellow, red int64
			switch models.StateColor(response.Dispstatus) {
			case models.StateColorGray:
				gray = 1
			case models.StateColorGreen:
				green = 1
			case models.StateColorYellow:
				yellow = 1
			case models.StateColorRed:
				red = 1
			}

			s.mb.RecordSapnetweaverSystemInstanceAvailabilityDataPoint(now, gray, response.Hostname, int64(response.InstanceNr), feature, metadata.AttributeControlStateGray)
			s.mb.RecordSapnetweaverSystemInstanceAvailabilityDataPoint(now, green, response.Hostname, int64(response.InstanceNr), feature, metadata.AttributeControlStateGreen)
			s.mb.RecordSapnetweaverSystemInstanceAvailabilityDataPoint(now, yellow, response.Hostname, int64(response.InstanceNr), feature, metadata.AttributeControlStateYellow)
			s.mb.RecordSapnetweaverSystemInstanceAvailabilityDataPoint(now, red, response.Hostname, int64(response.InstanceNr), feature, metadata.AttributeControlStateRed)
		}
	}
}

func (s *sapNetweaverScraper) recordSapnetweaverProcessAvailabilityDataPoint(now pcommon.Timestamp, processListResponse *models.GetProcessListResponse, errs *scrapererror.ScrapeErrors) {
	metricName := "Process Availability"
	if processListResponse.Process.Item == nil {
		err := formatErrorMsg(metricName, "", errValueNotFound)
		errs.AddPartial(1, err)
		return
	}

	for _, response := range processListResponse.Process.Item {
		var gray, green, yellow, red int64
		switch models.StateColor(*response.Dispstatus) {
		case models.StateColorGray:
			gray = 1
		case models.StateColorGreen:
			green = 1
		case models.StateColorYellow:
			yellow = 1
		case models.StateColorRed:
			red = 1
		}
		s.mb.RecordSapnetweaverProcessAvailabilityDataPoint(now, gray, response.Name, response.Description, metadata.AttributeControlStateGray)
		s.mb.RecordSapnetweaverProcessAvailabilityDataPoint(now, green, response.Name, response.Description, metadata.AttributeControlStateGreen)
		s.mb.RecordSapnetweaverProcessAvailabilityDataPoint(now, yellow, response.Name, response.Description, metadata.AttributeControlStateYellow)
		s.mb.RecordSapnetweaverProcessAvailabilityDataPoint(now, red, response.Name, response.Description, metadata.AttributeControlStateRed)
	}
}

func (s *sapNetweaverScraper) recordSapnetweaverWorkProcessJobAbortedCountDataPoint(now pcommon.Timestamp, alertTreeResponse map[string]string, errs *scrapererror.ScrapeErrors) {
	metricName := "AbortedJobs"
	val, ok := alertTreeResponse[metricName]
	if !ok {
		errs.AddPartial(1, fmt.Errorf(collectMetricError, metricName, errValueNotFound))
		return
	}

	var gray, green, yellow, red int64
	switch models.StateColor(models.StateColor(val)) {
	case models.StateColorGray:
		gray = 1
	case models.StateColorGreen:
		green = 1
	case models.StateColorYellow:
		yellow = 1
	case models.StateColorRed:
		red = 1
	}

	s.mb.RecordSapnetweaverWorkProcessJobAbortedStatusDataPoint(now, gray, metadata.AttributeControlStateGray)
	s.mb.RecordSapnetweaverWorkProcessJobAbortedStatusDataPoint(now, green, metadata.AttributeControlStateGreen)
	s.mb.RecordSapnetweaverWorkProcessJobAbortedStatusDataPoint(now, yellow, metadata.AttributeControlStateYellow)
	s.mb.RecordSapnetweaverWorkProcessJobAbortedStatusDataPoint(now, red, metadata.AttributeControlStateRed)
}

// recordSapnetweaverDatabaseDialogRequestTimeDataPoint records the database dialog request time
func (s *sapNetweaverScraper) recordSapnetweaverDatabaseDialogRequestTimeDataPoint(now pcommon.Timestamp, alertTreeResponse map[string]string, errs *scrapererror.ScrapeErrors) {
	metricName := "DBRequestTime"
	val, err := parseResponse(metricName, "", alertTreeResponse)
	if err != nil {
		errs.AddPartial(1, err)
		return
	}

	err = s.mb.RecordSapnetweaverDatabaseDialogRequestTimeDataPoint(now, val)
	if err != nil {
		errs.AddPartial(1, err)
		return
	}
}

func (s *sapNetweaverScraper) recordSapnetweaverCPUUtilizationDataPoint(now pcommon.Timestamp, alertTreeResponse map[string]string, errs *scrapererror.ScrapeErrors) {
	metricName := "CPU_Utilization"
	val, err := parseResponse(metricName, "", alertTreeResponse)
	if err != nil {
		errs.AddPartial(1, err)
		return
	}

	err = s.mb.RecordSapnetweaverCPUUtilizationDataPoint(now, val)
	if err != nil {
		errs.AddPartial(1, err)
		return
	}
}

func (s *sapNetweaverScraper) recordSapnetweaverCPUSystemUtilizationDataPoint(now pcommon.Timestamp, alertTreeResponse map[string]string, errs *scrapererror.ScrapeErrors) {
	metricName := "System Utilization"
	val, err := parseResponse(metricName, "", alertTreeResponse)
	if err != nil {
		errs.AddPartial(1, err)
		return
	}

	err = s.mb.RecordSapnetweaverCPUSystemUtilizationDataPoint(now, val)
	if err != nil {
		errs.AddPartial(1, err)
		return
	}
}

func (s *sapNetweaverScraper) recordSapnetweaverMemorySwapSpaceUtilizationDataPoint(now pcommon.Timestamp, alertTreeResponse map[string]string, errs *scrapererror.ScrapeErrors) {
	// used this name for Percentage_Used which has a parent ID referencing Swap_Space
	metricName := "Swap_Space_Percentage_Used"
	val, err := parseResponse(metricName, "", alertTreeResponse)
	if err != nil {
		errs.AddPartial(1, err)
		return
	}

	err = s.mb.RecordSapnetweaverMemorySwapSpaceUtilizationDataPoint(now, val)
	if err != nil {
		errs.AddPartial(1, err)
		return
	}
}

func (s *sapNetweaverScraper) recordSapnetweaverHostMemoryVirtualSwapDataPoint(now pcommon.Timestamp, alertTreeResponse map[string]string, errs *scrapererror.ScrapeErrors) {
	metricName := "Memory Swapped Out"
	val, err := parseResponse(metricName, "", alertTreeResponse)
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

func (s *sapNetweaverScraper) recordSapnetweaverMemoryConfiguredDataPoint(now pcommon.Timestamp, alertTreeResponse map[string]string, errs *scrapererror.ScrapeErrors) {
	metricName := "Configured Memory"
	val, err := parseResponse(metricName, "", alertTreeResponse)
	if err != nil {
		errs.AddPartial(1, err)
		return
	}

	mbytes, err := strconv.ParseInt(val, 10, 64)
	if err != nil {
		errs.AddPartial(1, fmt.Errorf("failed to parse int64 for SapnetweaverMemoryConfigured, value was %v: %w", val, err))
		return
	}

	s.mb.RecordSapnetweaverMemoryConfiguredDataPoint(now, mbytes*int64(MBToBytes))
}

func (s *sapNetweaverScraper) recordSapnetweaverMemoryFreeDataPoint(now pcommon.Timestamp, alertTreeResponse map[string]string, errs *scrapererror.ScrapeErrors) {
	metricName := "Free Memory"
	val, err := parseResponse(metricName, "", alertTreeResponse)
	if err != nil {
		errs.AddPartial(1, err)
		return
	}

	mbytes, err := strconv.ParseInt(val, 10, 64)
	if err != nil {
		errs.AddPartial(1, fmt.Errorf("failed to parse int64 for SapnetweaverMemoryFree, value was %v: %w", val, err))
		return
	}

	s.mb.RecordSapnetweaverMemoryFreeDataPoint(now, mbytes*int64(MBToBytes))
}

func (s *sapNetweaverScraper) recordSapnetweaverHostMemoryVirtualOverheadDataPoint(now pcommon.Timestamp, alertTreeResponse map[string]string, errs *scrapererror.ScrapeErrors) {
	metricName := "Memory Overhead"
	val, err := parseResponse(metricName, "", alertTreeResponse)
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

// recordSapnetweaverWorkProcessActiveCountDataPoint
func (s *sapNetweaverScraper) recordSapnetweaverWorkProcessActiveCountDataPoint(now pcommon.Timestamp, systemWPTableResponse *models.ABAPGetSystemWPTableResponse, errs *scrapererror.ScrapeErrors) {
	metricName := "Active Work Processes"

	if systemWPTableResponse.Workprocess.Item == nil {
		err := formatErrorMsg(metricName, "", errValueNotFound)
		errs.AddPartial(1, err)
		return
	}

	if len(systemWPTableResponse.Workprocess.Item) == 1 {
		s.mb.RecordSapnetweaverWorkProcessActiveCountDataPoint(now, 1, systemWPTableResponse.Workprocess.Item[0].Instance, systemWPTableResponse.Workprocess.Item[0].Typ, systemWPTableResponse.Workprocess.Item[0].Status)
	}
	// We want to count the number of times the instance, type, and status are the same.
	// Since the values are sorted, we can iterate through the list and compare the current value to the next value.
	similarityCount := 1
	for i, j := 0, 1; j < len(systemWPTableResponse.Workprocess.Item); i, j = i+1, j+1 {
		if systemWPTableResponse.Workprocess.Item[i].Instance == systemWPTableResponse.Workprocess.Item[j].Instance &&
			systemWPTableResponse.Workprocess.Item[i].Typ == systemWPTableResponse.Workprocess.Item[j].Typ &&
			systemWPTableResponse.Workprocess.Item[i].Status == systemWPTableResponse.Workprocess.Item[j].Status {
			similarityCount++
		} else {
			s.mb.RecordSapnetweaverWorkProcessActiveCountDataPoint(now, int64(similarityCount), systemWPTableResponse.Workprocess.Item[i].Instance, systemWPTableResponse.Workprocess.Item[i].Typ, systemWPTableResponse.Workprocess.Item[i].Status)
			similarityCount = 1
		}
	}

	// Record the last value
	if systemWPTableResponse.Workprocess.Item[len(systemWPTableResponse.Workprocess.Item)-2].Instance == systemWPTableResponse.Workprocess.Item[len(systemWPTableResponse.Workprocess.Item)-1].Instance &&
		systemWPTableResponse.Workprocess.Item[len(systemWPTableResponse.Workprocess.Item)-2].Typ == systemWPTableResponse.Workprocess.Item[len(systemWPTableResponse.Workprocess.Item)-1].Typ &&
		systemWPTableResponse.Workprocess.Item[len(systemWPTableResponse.Workprocess.Item)-2].Status == systemWPTableResponse.Workprocess.Item[len(systemWPTableResponse.Workprocess.Item)-1].Status {
		similarityCount++
	} else {
		similarityCount = 1
	}

	s.mb.RecordSapnetweaverWorkProcessActiveCountDataPoint(now, int64(similarityCount), systemWPTableResponse.Workprocess.Item[len(systemWPTableResponse.Workprocess.Item)-1].Instance, systemWPTableResponse.Workprocess.Item[len(systemWPTableResponse.Workprocess.Item)-1].Typ, systemWPTableResponse.Workprocess.Item[len(systemWPTableResponse.Workprocess.Item)-1].Status)
}

// recordSapnetweaverAbapRfcCountDataPoint
func (s *sapNetweaverScraper) recordSapnetweaverAbapRfcCountDataPoint(now pcommon.Timestamp, rfcTable string) {
	rfcConnectionTable := parseRfcConnectionsTable(rfcTable)
	for rfcType, count := range rfcConnectionTable {
		s.mb.RecordSapnetweaverAbapRfcCountDataPoint(now, count, rfcType)
	}
}

func parseRfcConnectionsTable(rfcTable string) map[string]int64 {
	rfcTypes := map[string]int64{}
	if strings.Contains(rfcTable, "Communication Table is empty") {
		return rfcTypes
	}

	for _, line := range strings.Split(rfcTable, "\n") {
		// select content between | and |, but not the header with  --- header fields, i.e. type
		if strings.Contains(line, "|") && !strings.Contains(line, "---") && !strings.Contains(line, "type") {
			fields := strings.Split(line, "|")
			if len(fields) >= 3 {
				// add string type to map count
				typeField := strings.TrimSpace(fields[2])
				rfcTypes[typeField]++
			}
		}
	}

	return rfcTypes
}

// recordSapnetweaverAbapSessionCountDataPoint
func (s *sapNetweaverScraper) recordSapnetweaverAbapSessionCountDataPoint(now pcommon.Timestamp, sessionTable string) {
	sessionsTable := parseSessionTable(sessionTable)
	for sessionType, count := range sessionsTable {
		s.mb.RecordSapnetweaverAbapSessionCountDataPoint(now, count, sessionType)
	}
}

func parseSessionTable(sessionTable string) map[string]int64 {
	sessionTypes := map[string]int64{}

	for _, line := range strings.Split(sessionTable, "\n") {
		// select content between | and |, but not the header with  --- header fields, i.e. type
		if strings.Contains(line, "|") && !strings.Contains(line, "---") && !strings.Contains(line, "Type") {
			fields := strings.Split(line, "|")
			if len(fields) >= 2 {
				// add string type to map count
				typeField := strings.TrimSpace(fields[1])
				sessionTypes[typeField]++
			}
		}
	}

	return sessionTypes
}

// recordSapnetweaverQueueCountDataPoint
func (s *sapNetweaverScraper) recordSapnetweaverQueueDataPoints(now pcommon.Timestamp, queueStatistic *models.GetQueueStatisticResponse, errs *scrapererror.ScrapeErrors) {
	if queueStatistic.Queue.Item == nil {
		err := formatErrorMsg("Queue count, peak and max", "", errValueNotFound)
		errs.AddPartial(1, err)
		return
	}
	for _, queue := range queueStatistic.Queue.Item {
		s.mb.RecordSapnetweaverQueueCountDataPoint(now, int64(queue.Now), queue.Typ)
		s.mb.RecordSapnetweaverQueuePeakCountDataPoint(now, int64(queue.High), queue.Typ)
		s.mb.RecordSapnetweaverQueueMaxCountDataPoint(now, int64(queue.Max), queue.Typ)
	}
}

func (s *sapNetweaverScraper) recordSapnetweaverSpoolRequestErrorCountDataPoint(now pcommon.Timestamp, alertTreeResponse map[string]string, errs *scrapererror.ScrapeErrors) {
	metricName := "ErrorsInWpSPO"
	val, err := parseResponse(metricName, "", alertTreeResponse)
	if err != nil {
		errs.AddPartial(1, err)
		return
	}

	err = s.mb.RecordSapnetweaverSpoolRequestErrorCountDataPoint(now, val)
	if err != nil {
		errs.AddPartial(1, err)
		return
	}
}

func (s *sapNetweaverScraper) recordSapnetweaverLocksDataPoints(now pcommon.Timestamp, enqStatisticsResponse *models.EnqGetStatisticResponse, errs *scrapererror.ScrapeErrors) {
	if enqStatisticsResponse.LocksNow != nil {
		s.mb.RecordSapnetweaverLocksEnqueueCurrentCountDataPoint(now, int64(*enqStatisticsResponse.LocksNow))
	} else {
		metricName := "LocksNow"
		err := formatErrorMsg(metricName, "", errValueNotFound)
		errs.AddPartial(1, err)
	}
	if enqStatisticsResponse.LocksHigh != nil {
		s.mb.RecordSapnetweaverLocksEnqueueHighCountDataPoint(now, int64(*enqStatisticsResponse.LocksHigh))
	} else {
		metricName := "LocksHigh"
		err := formatErrorMsg(metricName, "", errValueNotFound)
		errs.AddPartial(1, err)
	}
	if enqStatisticsResponse.LocksMax != nil {
		s.mb.RecordSapnetweaverLocksEnqueueMaxCountDataPoint(now, int64(*enqStatisticsResponse.LocksMax))
	} else {
		metricName := "LocksMax"
		err := formatErrorMsg(metricName, "", errValueNotFound)
		errs.AddPartial(1, err)
	}
	if enqStatisticsResponse.DequeueErrors != nil {
		s.mb.RecordSapnetweaverLocksDequeueErrorsCountDataPoint(now, int64(*enqStatisticsResponse.DequeueErrors))
	} else {
		metricName := "DequeueErrors"
		err := formatErrorMsg(metricName, "", errValueNotFound)
		errs.AddPartial(1, err)
	}
	if enqStatisticsResponse.EnqueueErrors != nil {
		s.mb.RecordSapnetweaverLocksEnqueueErrorsCountDataPoint(now, int64(*enqStatisticsResponse.EnqueueErrors))
	} else {
		metricName := "EnqueueErrors"
		err := formatErrorMsg(metricName, "", errValueNotFound)
		errs.AddPartial(1, err)
	}
	if enqStatisticsResponse.LockTime != nil {
		s.mb.RecordSapnetweaverLocksEnqueueLockTimeDataPoint(now, int64(*enqStatisticsResponse.LockTime))
	} else {
		metricName := "LockTime"
		err := formatErrorMsg(metricName, "", errValueNotFound)
		errs.AddPartial(1, err)
	}
	if enqStatisticsResponse.LockWaitTime != nil {
		s.mb.RecordSapnetweaverLocksEnqueueLockWaitTimeDataPoint(now, int64(*enqStatisticsResponse.LockWaitTime))
	} else {
		metricName := "LockWaitTime"
		err := formatErrorMsg(metricName, "", errValueNotFound)
		errs.AddPartial(1, err)
	}
}

func (s *sapNetweaverScraper) recordSapnetweaverShortDumpsCountDataPoint(now pcommon.Timestamp, alertTreeResponse map[string]string, errs *scrapererror.ScrapeErrors) {
	metricName := "Shortdumps Frequency"
	val, err := parseResponse(metricName, "", alertTreeResponse)
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

func (s *sapNetweaverScraper) recordSapnetweaverSessionsHTTPCountDataPoint(now pcommon.Timestamp, alertTreeResponse map[string]string, errs *scrapererror.ScrapeErrors) {
	metricName := "CurrentHttpSessions"
	val, err := parseResponse(metricName, "", alertTreeResponse)
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
	val, err := parseResponse(metricName, "", alertTreeResponse)
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

func (s *sapNetweaverScraper) recordSapnetweaverSessionsBrowserCountDataPoint(now pcommon.Timestamp, alertTreeResponse map[string]string, errs *scrapererror.ScrapeErrors) {
	metricName := "Browser Sessions"
	val, err := parseResponse(metricName, "", alertTreeResponse)
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

func (s *sapNetweaverScraper) recordSapnetweaverSessionsWebCountDataPoint(now pcommon.Timestamp, alertTreeResponse map[string]string, errs *scrapererror.ScrapeErrors) {
	metricName := "Web Sessions"
	val, err := parseResponse(metricName, "", alertTreeResponse)
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

func (s *sapNetweaverScraper) recordSapnetweaverSessionsEjbCountDataPoint(now pcommon.Timestamp, alertTreeResponse map[string]string, errs *scrapererror.ScrapeErrors) {
	metricName := "EJB Sessions"
	val, err := parseResponse(metricName, "", alertTreeResponse)
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

func (s *sapNetweaverScraper) recordSapnetweaverSessionCountDataPoint(now pcommon.Timestamp, alertTreeResponse map[string]string, errs *scrapererror.ScrapeErrors) {
	metricName := "Number of Sessions"
	val, err := parseResponse(metricName, "", alertTreeResponse)
	if err != nil {
		errs.AddPartial(1, err)
		return
	}

	err = s.mb.RecordSapnetweaverSessionCountDataPoint(now, val)
	if err != nil {
		errs.AddPartial(1, err)
		return
	}
}

func (s *sapNetweaverScraper) recordSapnetweaverAbapUpdateStatusDataPoint(now pcommon.Timestamp, alertTreeResponse map[string]string, errs *scrapererror.ScrapeErrors) {
	metricName := "AbapErrorInUpdate"
	val, ok := alertTreeResponse[metricName]
	if !ok {
		errs.AddPartial(1, fmt.Errorf(collectMetricError, metricName, errValueNotFound))
		return
	}
	var gray, green, yellow, red int64
	switch models.StateColor(models.StateColor(val)) {
	case models.StateColorGray:
		gray = 1
	case models.StateColorGreen:
		green = 1
	case models.StateColorYellow:
		yellow = 1
	case models.StateColorRed:
		red = 1
	}
	s.mb.RecordSapnetweaverAbapUpdateStatusDataPoint(now, gray, metadata.AttributeControlStateGray)
	s.mb.RecordSapnetweaverAbapUpdateStatusDataPoint(now, green, metadata.AttributeControlStateGreen)
	s.mb.RecordSapnetweaverAbapUpdateStatusDataPoint(now, yellow, metadata.AttributeControlStateYellow)
	s.mb.RecordSapnetweaverAbapUpdateStatusDataPoint(now, red, metadata.AttributeControlStateRed)
}

func (s *sapNetweaverScraper) recordSapnetweaverResponseDurationDataPoint(now pcommon.Timestamp, alertTreeResponse map[string]string, errs *scrapererror.ScrapeErrors) {
	s.recordSapnetweaverResponseDurationDataPointDialog(now, alertTreeResponse, errs)
	s.recordSapnetweaverResponseDurationDataPointDialogRFC(now, alertTreeResponse, errs)
	s.recordSapnetweaverResponseDurationDataPointTransaction(now, alertTreeResponse, errs)
	s.recordSapnetweaverResponseDurationDataPointHTTP(now, alertTreeResponse, errs)
}

func (s *sapNetweaverScraper) recordSapnetweaverResponseDurationDataPointDialog(now pcommon.Timestamp, alertTreeResponse map[string]string, errs *scrapererror.ScrapeErrors) {
	metricName := "ResponseTimeDialog"
	val, err := parseResponse(metricName, metadata.AttributeResponseTypeDialog.String(), alertTreeResponse)
	if err != nil {
		errs.AddPartial(1, err)
		return
	}

	err = s.mb.RecordSapnetweaverResponseDurationDataPoint(now, val, metadata.AttributeResponseTypeDialog)
	if err != nil {
		errs.AddPartial(1, err)
		return
	}
}

func (s *sapNetweaverScraper) recordSapnetweaverResponseDurationDataPointDialogRFC(now pcommon.Timestamp, alertTreeResponse map[string]string, errs *scrapererror.ScrapeErrors) {
	metricName := "ResponseTimeDialogRFC"
	val, err := parseResponse(metricName, metadata.AttributeResponseTypeDialogRFC.String(), alertTreeResponse)
	if err != nil {
		errs.AddPartial(1, err)
		return
	}

	err = s.mb.RecordSapnetweaverResponseDurationDataPoint(now, val, metadata.AttributeResponseTypeDialogRFC)
	if err != nil {
		errs.AddPartial(1, err)
		return
	}
}

func (s *sapNetweaverScraper) recordSapnetweaverResponseDurationDataPointTransaction(now pcommon.Timestamp, alertTreeResponse map[string]string, errs *scrapererror.ScrapeErrors) {
	metricName := "ResponseTime(StandardTran.)"
	val, err := parseResponse(metricName, metadata.AttributeResponseTypeTransaction.String(), alertTreeResponse)
	if err != nil {
		errs.AddPartial(1, err)
		return
	}

	err = s.mb.RecordSapnetweaverResponseDurationDataPoint(now, val, metadata.AttributeResponseTypeTransaction)
	if err != nil {
		errs.AddPartial(1, err)
		return
	}
}

func (s *sapNetweaverScraper) recordSapnetweaverResponseDurationDataPointHTTP(now pcommon.Timestamp, alertTreeResponse map[string]string, errs *scrapererror.ScrapeErrors) {
	metricName := "ResponseTimeHTTP"
	val, err := parseResponse(metricName, metadata.AttributeResponseTypeHttp.String(), alertTreeResponse)
	if err != nil {
		errs.AddPartial(1, err)
		return
	}

	err = s.mb.RecordSapnetweaverResponseDurationDataPoint(now, val, metadata.AttributeResponseTypeHttp)
	if err != nil {
		errs.AddPartial(1, err)
		return
	}
}

func (s *sapNetweaverScraper) recordSapnetweaverRequestCountDataPoint(now pcommon.Timestamp, alertTreeResponse map[string]string, errs *scrapererror.ScrapeErrors) {
	metricName := "StatNoOfRequests"
	val, err := parseResponse(metricName, "", alertTreeResponse)
	if err != nil {
		errs.AddPartial(1, err)
		return
	}

	err = s.mb.RecordSapnetweaverRequestCountDataPoint(now, val)
	if err != nil {
		errs.AddPartial(1, err)
		return
	}
}

func (s *sapNetweaverScraper) recordSapnetweaverRequestTimeoutCountDataPoint(now pcommon.Timestamp, alertTreeResponse map[string]string, errs *scrapererror.ScrapeErrors) {
	metricName := "StatNoOfTimeouts"
	val, err := parseResponse(metricName, "", alertTreeResponse)
	if err != nil {
		errs.AddPartial(1, err)
		return
	}

	err = s.mb.RecordSapnetweaverRequestTimeoutCountDataPoint(now, val)
	if err != nil {
		errs.AddPartial(1, err)
		return
	}
}

func (s *sapNetweaverScraper) recordSapnetweaverConnectionErrorCountDataPoint(now pcommon.Timestamp, alertTreeResponse map[string]string, errs *scrapererror.ScrapeErrors) {
	metricName := "StatNoOfConnectErrors"
	val, err := parseResponse(metricName, "", alertTreeResponse)
	if err != nil {
		errs.AddPartial(1, err)
		return
	}

	err = s.mb.RecordSapnetweaverConnectionErrorCountDataPoint(now, val)
	if err != nil {
		errs.AddPartial(1, err)
		return
	}
}

func (s *sapNetweaverScraper) recordSapnetweaverCacheHitsDataPoint(now pcommon.Timestamp, alertTreeResponse map[string]string, errs *scrapererror.ScrapeErrors) {
	metricName := "CacheHits"
	val, err := parseResponse(metricName, "", alertTreeResponse)
	if err != nil {
		errs.AddPartial(1, err)
		return
	}

	err = s.mb.RecordSapnetweaverCacheHitsDataPoint(now, val)
	if err != nil {
		errs.AddPartial(1, err)
		return
	}
}

func (s *sapNetweaverScraper) recordSapnetweaverCacheEvictionsDataPoint(now pcommon.Timestamp, alertTreeResponse map[string]string, errs *scrapererror.ScrapeErrors) {
	metricName := "EvictedEntries"
	val, err := parseResponse(metricName, "", alertTreeResponse)
	if err != nil {
		errs.AddPartial(1, err)
		return
	}

	err = s.mb.RecordSapnetweaverCacheEvictionsDataPoint(now, val)
	if err != nil {
		errs.AddPartial(1, err)
		return
	}
}

func (s *sapNetweaverScraper) recordSapnetweaverHostSpoolListUtilizationDataPoint(now pcommon.Timestamp, alertTreeResponse map[string]string, errs *scrapererror.ScrapeErrors) {
	metricName := "HostspoolListUsed"
	val, err := parseResponse(metricName, "", alertTreeResponse)
	if err != nil {
		errs.AddPartial(1, err)
		return
	}

	err = s.mb.RecordSapnetweaverHostSpoolListUtilizationDataPoint(now, val)
	if err != nil {
		errs.AddPartial(1, err)
		return
	}
}

func parseResponse(metricName string, attributeName string, alertTreeResponse map[string]string) (string, error) {
	val, ok := alertTreeResponse[metricName]
	if !ok {
		return "", formatErrorMsg(metricName, attributeName, errValueNotFound)
	}

	if strings.Contains(val, "-") {
		return "", formatErrorMsg(metricName, attributeName, errValueHyphen)
	}
	return val, nil
}

func formatErrorMsg(metricName string, attributeName string, errValue error) error {
	if attributeName == "" {
		return fmt.Errorf(collectMetricError, metricName, errValue)
	}

	return fmt.Errorf(collectMetricErrorWithAttribute, metricName, attributeName, errValue)
}
