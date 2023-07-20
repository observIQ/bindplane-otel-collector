// Copyright observIQ, Inc.
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

package m365receiver // import "github.com/observiq/bindplane-agent/receiver/m365receiver"

import (
	"context"
	"strconv"
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/receiver"
	"go.uber.org/zap"

	"github.com/observiq/bindplane-agent/receiver/m365receiver/internal/metadata"
)

type reportPair struct {
	columns map[string]int //relevant metric name and index for csv file

	endpoint string                                                                     //unique report endpoint
	indexes  map[int]func(mb *metadata.MetricsBuilder, ts pcommon.Timestamp, val int64) //map of csv file index and recordMetric func
}

var reports = []reportPair{
	{
		//columns:  map[string]int{"sharepoint.files": 2, "sharepoint.files.active": 3},
		endpoint: "getSharePointSiteUsageFileCounts(period='D7')",
		indexes: map[int]func(mb *metadata.MetricsBuilder, ts pcommon.Timestamp, val int64){
			2: func(mb *metadata.MetricsBuilder, ts pcommon.Timestamp, val int64) {
				mb.RecordM365SharepointFilesCountDataPoint(ts, val)
			},
			3: func(mb *metadata.MetricsBuilder, ts pcommon.Timestamp, val int64) {
				mb.RecordM365SharepointFilesActiveCountDataPoint(ts, val)
			},
		},
	},
	{
		//columns:  map[string]int{"sharepoint.sites.active": 3},
		endpoint: "getSharePointSiteUsageSiteCounts(period='D7')",
		indexes: map[int]func(mb *metadata.MetricsBuilder, ts pcommon.Timestamp, val int64){
			3: func(mb *metadata.MetricsBuilder, ts pcommon.Timestamp, val int64) {
				mb.RecordM365SharepointSitesActiveCountDataPoint(ts, val)
			},
		},
	},
	{
		//columns:  map[string]int{"sharepoint.pages.viewed": 2},
		endpoint: "getSharePointSiteUsagePages(period='D7')",
		indexes: map[int]func(mb *metadata.MetricsBuilder, ts pcommon.Timestamp, val int64){
			2: func(mb *metadata.MetricsBuilder, ts pcommon.Timestamp, val int64) {
				mb.RecordM365SharepointPagesViewedCountDataPoint(ts, val)
			},
		},
	},
	{
		//columns:  map[string]int{"sharepoint.pages.unique": 1},
		endpoint: "getSharePointActivityPages(period='D7')",
		indexes: map[int]func(mb *metadata.MetricsBuilder, ts pcommon.Timestamp, val int64){
			1: func(mb *metadata.MetricsBuilder, ts pcommon.Timestamp, val int64) {
				mb.RecordM365SharepointPagesUniqueCountDataPoint(ts, val)
			},
		},
	},
	{
		//columns:  map[string]int{"sharepoint.site.storage": 2},
		endpoint: "getSharePointSiteUsageStorage(period='D7')",
		indexes: map[int]func(mb *metadata.MetricsBuilder, ts pcommon.Timestamp, val int64){
			2: func(mb *metadata.MetricsBuilder, ts pcommon.Timestamp, val int64) {
				mb.RecordM365SharepointSiteStorageUsedDataPoint(ts, val)
			},
		},
	},
	{
		//columns:  map[string]int{"device.web": 1, "device.android": 3, "device.ios": 4, "device.mac": 5, "device.windows": 6, "device.chromeos": 7, "device.linux": 8},
		endpoint: "getTeamsDeviceUsageDistributionUserCounts(period='D7')",
		indexes: map[int]func(mb *metadata.MetricsBuilder, ts pcommon.Timestamp, val int64){
			1: func(mb *metadata.MetricsBuilder, ts pcommon.Timestamp, val int64) {
				mb.RecordM365TeamsDeviceUsageUsersDataPoint(ts, val, metadata.AttributeTeamsDevicesWeb)
			},
			3: func(mb *metadata.MetricsBuilder, ts pcommon.Timestamp, val int64) {
				mb.RecordM365TeamsDeviceUsageUsersDataPoint(ts, val, metadata.AttributeTeamsDevicesAndroid)
			},
			4: func(mb *metadata.MetricsBuilder, ts pcommon.Timestamp, val int64) {
				mb.RecordM365TeamsDeviceUsageUsersDataPoint(ts, val, metadata.AttributeTeamsDevicesIOS)
			},
			5: func(mb *metadata.MetricsBuilder, ts pcommon.Timestamp, val int64) {
				mb.RecordM365TeamsDeviceUsageUsersDataPoint(ts, val, metadata.AttributeTeamsDevicesMac)
			},
			6: func(mb *metadata.MetricsBuilder, ts pcommon.Timestamp, val int64) {
				mb.RecordM365TeamsDeviceUsageUsersDataPoint(ts, val, metadata.AttributeTeamsDevicesWindows)
			},
			7: func(mb *metadata.MetricsBuilder, ts pcommon.Timestamp, val int64) {
				mb.RecordM365TeamsDeviceUsageUsersDataPoint(ts, val, metadata.AttributeTeamsDevicesChromeOS)
			},
			8: func(mb *metadata.MetricsBuilder, ts pcommon.Timestamp, val int64) {
				mb.RecordM365TeamsDeviceUsageUsersDataPoint(ts, val, metadata.AttributeTeamsDevicesLinux)
			},
		},
	},
	{
		//columns:  map[string]int{"teams.message.team": 2, "teams.message.private": 5, "teams.calls": 6, "teams.meetings": 7},
		endpoint: "getTeamsUserActivityCounts(period='D7')",
		indexes: map[int]func(mb *metadata.MetricsBuilder, ts pcommon.Timestamp, val int64){
			2: func(mb *metadata.MetricsBuilder, ts pcommon.Timestamp, val int64) {
				mb.RecordM365TeamsMessagesTeamCountDataPoint(ts, val)
			},
			5: func(mb *metadata.MetricsBuilder, ts pcommon.Timestamp, val int64) {
				mb.RecordM365TeamsMessagesPrivateCountDataPoint(ts, val)
			},
			6: func(mb *metadata.MetricsBuilder, ts pcommon.Timestamp, val int64) {
				mb.RecordM365TeamsCallsCountDataPoint(ts, val)
			},
			7: func(mb *metadata.MetricsBuilder, ts pcommon.Timestamp, val int64) {
				mb.RecordM365TeamsMeetingsCountDataPoint(ts, val)
			},
		},
	},
	{
		//columns:  map[string]int{"onedrive.files": 2, "onedrive.files.active": 3},
		endpoint: "getOneDriveUsageFileCounts(period='D7')",
		indexes: map[int]func(mb *metadata.MetricsBuilder, ts pcommon.Timestamp, val int64){
			2: func(mb *metadata.MetricsBuilder, ts pcommon.Timestamp, val int64) {
				mb.RecordM365OnedriveFilesCountDataPoint(ts, val)
			},
			3: func(mb *metadata.MetricsBuilder, ts pcommon.Timestamp, val int64) {
				mb.RecordM365OnedriveFilesActiveCountDataPoint(ts, val)
			},
		},
	},
	{
		//columns:  map[string]int{"onedrive.activity.view_edit": 1, "onedrive.activity.synced": 2, "onedrive.activity.internal": 3, "onedrive.activity.external": 4},
		endpoint: "getOneDriveActivityUserCounts(period='D7')",
		indexes: map[int]func(mb *metadata.MetricsBuilder, ts pcommon.Timestamp, val int64){
			1: func(mb *metadata.MetricsBuilder, ts pcommon.Timestamp, val int64) {
				mb.RecordM365OnedriveUserActivityCountDataPoint(ts, val, metadata.AttributeOnedriveActivityViewEdit)
			},
			2: func(mb *metadata.MetricsBuilder, ts pcommon.Timestamp, val int64) {
				mb.RecordM365OnedriveUserActivityCountDataPoint(ts, val, metadata.AttributeOnedriveActivitySynced)
			},
			3: func(mb *metadata.MetricsBuilder, ts pcommon.Timestamp, val int64) {
				mb.RecordM365OnedriveUserActivityCountDataPoint(ts, val, metadata.AttributeOnedriveActivityInternalShare)
			},
			4: func(mb *metadata.MetricsBuilder, ts pcommon.Timestamp, val int64) {
				mb.RecordM365OnedriveUserActivityCountDataPoint(ts, val, metadata.AttributeOnedriveActivityExternalShare)
			},
		},
	},
	{
		//columns:  map[string]int{"outlook.mailboxes.active": 2},
		endpoint: "getMailboxUsageMailboxCounts(period='D7')",
		indexes: map[int]func(mb *metadata.MetricsBuilder, ts pcommon.Timestamp, val int64){
			2: func(mb *metadata.MetricsBuilder, ts pcommon.Timestamp, val int64) {
				mb.RecordM365OutlookMailboxesActiveCountDataPoint(ts, val)
			},
		},
	},
	{
		//columns:  map[string]int{"outlook.read": 3, "outlook.sent": 1, "outlook.received": 2},
		endpoint: "getEmailActivityCounts(period='D7')",
		indexes: map[int]func(mb *metadata.MetricsBuilder, ts pcommon.Timestamp, val int64){
			1: func(mb *metadata.MetricsBuilder, ts pcommon.Timestamp, val int64) {
				mb.RecordM365OutlookEmailActivityCountDataPoint(ts, val, metadata.AttributeOutlookActivitySent)
			},
			2: func(mb *metadata.MetricsBuilder, ts pcommon.Timestamp, val int64) {
				mb.RecordM365OutlookEmailActivityCountDataPoint(ts, val, metadata.AttributeOutlookActivityReceived)
			},
			3: func(mb *metadata.MetricsBuilder, ts pcommon.Timestamp, val int64) {
				mb.RecordM365OutlookEmailActivityCountDataPoint(ts, val, metadata.AttributeOutlookActivityRead)
			},
		},
	},
	{
		//columns:  map[string]int{"outlook.storage": 1},
		endpoint: "getMailboxUsageStorage(period='D7')",
		indexes: map[int]func(mb *metadata.MetricsBuilder, ts pcommon.Timestamp, val int64){
			1: func(mb *metadata.MetricsBuilder, ts pcommon.Timestamp, val int64) {
				mb.RecordM365OutlookStorageUsedDataPoint(ts, val)
			},
		},
	},
	{
		//columns:  map[string]int{"outlook.pop3": 7, "outlook.imap4": 8, "outlook.smtp": 9, "outlook.windows": 3, "outlook.mac": 2, "outlook.web": 6, "outlook.mobile": 4, "outlook.other_mobile": 5},
		endpoint: "getEmailAppUsageAppsUserCounts(period='D7')",
		indexes: map[int]func(mb *metadata.MetricsBuilder, ts pcommon.Timestamp, val int64){
			2: func(mb *metadata.MetricsBuilder, ts pcommon.Timestamp, val int64) {
				mb.RecordM365OutlookAppUserCountDataPoint(ts, val, metadata.AttributeOutlookAppsMac)
			},
			3: func(mb *metadata.MetricsBuilder, ts pcommon.Timestamp, val int64) {
				mb.RecordM365OutlookAppUserCountDataPoint(ts, val, metadata.AttributeOutlookAppsWindows)
			},
			4: func(mb *metadata.MetricsBuilder, ts pcommon.Timestamp, val int64) {
				mb.RecordM365OutlookAppUserCountDataPoint(ts, val, metadata.AttributeOutlookAppsMobile)
			},
			5: func(mb *metadata.MetricsBuilder, ts pcommon.Timestamp, val int64) {
				mb.RecordM365OutlookAppUserCountDataPoint(ts, val, metadata.AttributeOutlookAppsOtherMobile)
			},
			6: func(mb *metadata.MetricsBuilder, ts pcommon.Timestamp, val int64) {
				mb.RecordM365OutlookAppUserCountDataPoint(ts, val, metadata.AttributeOutlookAppsWeb)
			},
			7: func(mb *metadata.MetricsBuilder, ts pcommon.Timestamp, val int64) {
				mb.RecordM365OutlookAppUserCountDataPoint(ts, val, metadata.AttributeOutlookAppsPop3)
			},
			8: func(mb *metadata.MetricsBuilder, ts pcommon.Timestamp, val int64) {
				mb.RecordM365OutlookAppUserCountDataPoint(ts, val, metadata.AttributeOutlookAppsImap4)
			},
			9: func(mb *metadata.MetricsBuilder, ts pcommon.Timestamp, val int64) {
				mb.RecordM365OutlookAppUserCountDataPoint(ts, val, metadata.AttributeOutlookAppsSmtp)
			},
		},
	},
	{
		//columns:  map[string]int{"outlook.under_limit": 1, "outlook.warning": 2, "outlook.send_prohibited": 3, "outlook.send_receive_prohibited": 4, "outlook.indeterminate": 5},
		endpoint: "getMailboxUsageQuotaStatusMailboxCounts(period='D7')",
		indexes: map[int]func(mb *metadata.MetricsBuilder, ts pcommon.Timestamp, val int64){
			1: func(mb *metadata.MetricsBuilder, ts pcommon.Timestamp, val int64) {
				mb.RecordM365OutlookQuotaStatusCountDataPoint(ts, val, metadata.AttributeOutlookQuotasUnderLimit)
			},
			2: func(mb *metadata.MetricsBuilder, ts pcommon.Timestamp, val int64) {
				mb.RecordM365OutlookQuotaStatusCountDataPoint(ts, val, metadata.AttributeOutlookQuotasWarning)
			},
			3: func(mb *metadata.MetricsBuilder, ts pcommon.Timestamp, val int64) {
				mb.RecordM365OutlookQuotaStatusCountDataPoint(ts, val, metadata.AttributeOutlookQuotasSendProhibited)
			},
			4: func(mb *metadata.MetricsBuilder, ts pcommon.Timestamp, val int64) {
				mb.RecordM365OutlookQuotaStatusCountDataPoint(ts, val, metadata.AttributeOutlookQuotasSendReceiveProhibited)
			},
			5: func(mb *metadata.MetricsBuilder, ts pcommon.Timestamp, val int64) {
				mb.RecordM365OutlookQuotaStatusCountDataPoint(ts, val, metadata.AttributeOutlookQuotasIndeterminate)
			},
		},
	},
}

type mClient interface {
	GetCSV(ctx context.Context, endpoint string) ([]string, error)
	GetToken(ctx context.Context) error
	shutdown() error
}

type m365Scraper struct {
	settings component.TelemetrySettings
	logger   *zap.Logger
	cfg      *Config
	client   mClient
	mb       *metadata.MetricsBuilder
	root     string
}

func newM365Scraper(
	settings receiver.CreateSettings,
	cfg *Config,
) *m365Scraper {
	m := &m365Scraper{
		settings: settings.TelemetrySettings,
		logger:   settings.Logger,
		cfg:      cfg,
		mb:       metadata.NewMetricsBuilder(cfg.MetricsBuilderConfig, settings),
		root:     "https://graph.microsoft.com/v1.0/reports/",
	}
	return m
}

func (m *m365Scraper) start(ctx context.Context, host component.Host) error {
	httpClient, err := m.cfg.ToClient(host, m.settings)
	if err != nil {
		m.logger.Error("error creating HTTP client", zap.Error(err))
		return err
	}

	m.client = newM365Client(httpClient, m.cfg, "https://graph.microsoft.com/.default")

	err = m.client.GetToken(ctx)
	if err != nil {
		m.logger.Error("error creating authorization token", zap.Error(err))
		return err
	}

	return nil
}

func (m *m365Scraper) shutdown(_ context.Context) error {
	return m.client.shutdown()
}

// retrieves data, builds metrics & emits them
func (m *m365Scraper) scrape(ctx context.Context) (pmetric.Metrics, error) {
	err := m.getStats(ctx)
	if err != nil {
		//troubleshoot stale token
		if err.Error() == "access token invalid" {
			m.logger.Debug("possible stale token; attempting to regenerate")
			err = m.client.GetToken(ctx)
			if err != nil {
				//something went wrong with generating token.
				m.logger.Error("error creating authorization token", zap.Error(err))
				return pmetric.Metrics{}, err
			}
			//retry data retrieval with fresh token
			err = m.getStats(ctx)
			if err != nil {
				//not a stale access token error, unsure what is wrong
				m.logger.Error("unable to retrieve stats", zap.Error(err))
				return pmetric.Metrics{}, err
			}
		} else {
			//not an error with the access token
			m.logger.Error("error retrieving stats", zap.Error(err))
			return pmetric.Metrics{}, err
		}
	}

	return m.mb.Emit(metadata.WithM365TenantID(m.cfg.TenantID)), nil
}

func (m *m365Scraper) getStats(ctx context.Context) error {
	now := pcommon.NewTimestampFromTime(time.Now())

	for _, r := range reports {
		line, err := m.client.GetCSV(ctx, m.root+r.endpoint)
		if err != nil {
			return err
		}
		if len(line) == 0 {
			m.logger.Debug("no data available from endpoint and associated metrics: ", zap.String("endpoint", r.endpoint), zap.Error(err))
			continue
		}

		for k, v := range r.indexes {
			if line[k] == "" {
				continue
			}
			data, err := strconv.Atoi(line[k])
			if err != nil { // error converting string to int
				return err
			}
			v(m.mb, now, int64(data))
		}
	}

	return nil
}
