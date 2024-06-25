// Code generated by mdatagen. DO NOT EDIT.

package metadata

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/receiver/receivertest"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

type testDataSet int

const (
	testDataSetDefault testDataSet = iota
	testDataSetAll
	testDataSetNone
)

func TestMetricsBuilder(t *testing.T) {
	tests := []struct {
		name        string
		metricsSet  testDataSet
		resAttrsSet testDataSet
		expectEmpty bool
	}{
		{
			name: "default",
		},
		{
			name:        "all_set",
			metricsSet:  testDataSetAll,
			resAttrsSet: testDataSetAll,
		},
		{
			name:        "none_set",
			metricsSet:  testDataSetNone,
			resAttrsSet: testDataSetNone,
			expectEmpty: true,
		},
		{
			name:        "filter_set_include",
			resAttrsSet: testDataSetAll,
		},
		{
			name:        "filter_set_exclude",
			resAttrsSet: testDataSetAll,
			expectEmpty: true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			start := pcommon.Timestamp(1_000_000_000)
			ts := pcommon.Timestamp(1_000_001_000)
			observedZapCore, observedLogs := observer.New(zap.WarnLevel)
			settings := receivertest.NewNopCreateSettings()
			settings.Logger = zap.New(observedZapCore)
			mb := NewMetricsBuilder(loadMetricsBuilderConfig(t, test.name), settings, WithStartTime(start))

			expectedWarnings := 0

			assert.Equal(t, expectedWarnings, observedLogs.Len())

			defaultMetricsCount := 0
			allMetricsCount := 0

			defaultMetricsCount++
			allMetricsCount++
			mb.RecordM365OnedriveFilesActiveCountDataPoint(ts, 1)

			defaultMetricsCount++
			allMetricsCount++
			mb.RecordM365OnedriveFilesCountDataPoint(ts, 1)

			defaultMetricsCount++
			allMetricsCount++
			mb.RecordM365OnedriveUserActivityCountDataPoint(ts, 1, AttributeOnedriveActivityViewEdit)

			defaultMetricsCount++
			allMetricsCount++
			mb.RecordM365OutlookAppUserCountDataPoint(ts, 1, AttributeOutlookAppsPop3)

			defaultMetricsCount++
			allMetricsCount++
			mb.RecordM365OutlookEmailActivityCountDataPoint(ts, 1, AttributeOutlookActivityRead)

			defaultMetricsCount++
			allMetricsCount++
			mb.RecordM365OutlookMailboxesActiveCountDataPoint(ts, 1)

			defaultMetricsCount++
			allMetricsCount++
			mb.RecordM365OutlookQuotaStatusCountDataPoint(ts, 1, AttributeOutlookQuotasUnderLimit)

			defaultMetricsCount++
			allMetricsCount++
			mb.RecordM365OutlookStorageUsedDataPoint(ts, 1)

			defaultMetricsCount++
			allMetricsCount++
			mb.RecordM365SharepointFilesActiveCountDataPoint(ts, 1)

			defaultMetricsCount++
			allMetricsCount++
			mb.RecordM365SharepointFilesCountDataPoint(ts, 1)

			defaultMetricsCount++
			allMetricsCount++
			mb.RecordM365SharepointPagesUniqueCountDataPoint(ts, 1)

			defaultMetricsCount++
			allMetricsCount++
			mb.RecordM365SharepointPagesViewedCountDataPoint(ts, 1)

			defaultMetricsCount++
			allMetricsCount++
			mb.RecordM365SharepointSiteStorageUsedDataPoint(ts, 1)

			defaultMetricsCount++
			allMetricsCount++
			mb.RecordM365SharepointSitesActiveCountDataPoint(ts, 1)

			defaultMetricsCount++
			allMetricsCount++
			mb.RecordM365TeamsCallsCountDataPoint(ts, 1)

			defaultMetricsCount++
			allMetricsCount++
			mb.RecordM365TeamsDeviceUsageUsersDataPoint(ts, 1, AttributeTeamsDevicesAndroid)

			defaultMetricsCount++
			allMetricsCount++
			mb.RecordM365TeamsMeetingsCountDataPoint(ts, 1)

			defaultMetricsCount++
			allMetricsCount++
			mb.RecordM365TeamsMessagesPrivateCountDataPoint(ts, 1)

			defaultMetricsCount++
			allMetricsCount++
			mb.RecordM365TeamsMessagesTeamCountDataPoint(ts, 1)

			rb := mb.NewResourceBuilder()
			rb.SetM365TenantID("m365.tenant.id-val")
			res := rb.Emit()
			metrics := mb.Emit(WithResource(res))

			if test.expectEmpty {
				assert.Equal(t, 0, metrics.ResourceMetrics().Len())
				return
			}

			assert.Equal(t, 1, metrics.ResourceMetrics().Len())
			rm := metrics.ResourceMetrics().At(0)
			assert.Equal(t, res, rm.Resource())
			assert.Equal(t, 1, rm.ScopeMetrics().Len())
			ms := rm.ScopeMetrics().At(0).Metrics()
			if test.metricsSet == testDataSetDefault {
				assert.Equal(t, defaultMetricsCount, ms.Len())
			}
			if test.metricsSet == testDataSetAll {
				assert.Equal(t, allMetricsCount, ms.Len())
			}
			validatedMetrics := make(map[string]bool)
			for i := 0; i < ms.Len(); i++ {
				switch ms.At(i).Name() {
				case "m365.onedrive.files.active.count":
					assert.False(t, validatedMetrics["m365.onedrive.files.active.count"], "Found a duplicate in the metrics slice: m365.onedrive.files.active.count")
					validatedMetrics["m365.onedrive.files.active.count"] = true
					assert.Equal(t, pmetric.MetricTypeSum, ms.At(i).Type())
					assert.Equal(t, 1, ms.At(i).Sum().DataPoints().Len())
					assert.Equal(t, "The number of active files across the OneDrive in the last 7 days.", ms.At(i).Description())
					assert.Equal(t, "{files}", ms.At(i).Unit())
					assert.Equal(t, false, ms.At(i).Sum().IsMonotonic())
					assert.Equal(t, pmetric.AggregationTemporalityCumulative, ms.At(i).Sum().AggregationTemporality())
					dp := ms.At(i).Sum().DataPoints().At(0)
					assert.Equal(t, start, dp.StartTimestamp())
					assert.Equal(t, ts, dp.Timestamp())
					assert.Equal(t, pmetric.NumberDataPointValueTypeInt, dp.ValueType())
					assert.Equal(t, int64(1), dp.IntValue())
				case "m365.onedrive.files.count":
					assert.False(t, validatedMetrics["m365.onedrive.files.count"], "Found a duplicate in the metrics slice: m365.onedrive.files.count")
					validatedMetrics["m365.onedrive.files.count"] = true
					assert.Equal(t, pmetric.MetricTypeSum, ms.At(i).Type())
					assert.Equal(t, 1, ms.At(i).Sum().DataPoints().Len())
					assert.Equal(t, "The number of total files across the OneDrive for the last 7 days.", ms.At(i).Description())
					assert.Equal(t, "{files}", ms.At(i).Unit())
					assert.Equal(t, false, ms.At(i).Sum().IsMonotonic())
					assert.Equal(t, pmetric.AggregationTemporalityCumulative, ms.At(i).Sum().AggregationTemporality())
					dp := ms.At(i).Sum().DataPoints().At(0)
					assert.Equal(t, start, dp.StartTimestamp())
					assert.Equal(t, ts, dp.Timestamp())
					assert.Equal(t, pmetric.NumberDataPointValueTypeInt, dp.ValueType())
					assert.Equal(t, int64(1), dp.IntValue())
				case "m365.onedrive.user_activity.count":
					assert.False(t, validatedMetrics["m365.onedrive.user_activity.count"], "Found a duplicate in the metrics slice: m365.onedrive.user_activity.count")
					validatedMetrics["m365.onedrive.user_activity.count"] = true
					assert.Equal(t, pmetric.MetricTypeSum, ms.At(i).Type())
					assert.Equal(t, 1, ms.At(i).Sum().DataPoints().Len())
					assert.Equal(t, "The number of users who have interacted with a OneDrive file, by action, in the last 7 days.", ms.At(i).Description())
					assert.Equal(t, "{users}", ms.At(i).Unit())
					assert.Equal(t, false, ms.At(i).Sum().IsMonotonic())
					assert.Equal(t, pmetric.AggregationTemporalityCumulative, ms.At(i).Sum().AggregationTemporality())
					dp := ms.At(i).Sum().DataPoints().At(0)
					assert.Equal(t, start, dp.StartTimestamp())
					assert.Equal(t, ts, dp.Timestamp())
					assert.Equal(t, pmetric.NumberDataPointValueTypeInt, dp.ValueType())
					assert.Equal(t, int64(1), dp.IntValue())
					attrVal, ok := dp.Attributes().Get("activity")
					assert.True(t, ok)
					assert.EqualValues(t, "view_edit", attrVal.Str())
				case "m365.outlook.app.user.count":
					assert.False(t, validatedMetrics["m365.outlook.app.user.count"], "Found a duplicate in the metrics slice: m365.outlook.app.user.count")
					validatedMetrics["m365.outlook.app.user.count"] = true
					assert.Equal(t, pmetric.MetricTypeSum, ms.At(i).Type())
					assert.Equal(t, 1, ms.At(i).Sum().DataPoints().Len())
					assert.Equal(t, "The number of unique users per app over the period of time in the organization Outlook in the last 7 days.", ms.At(i).Description())
					assert.Equal(t, "{users}", ms.At(i).Unit())
					assert.Equal(t, false, ms.At(i).Sum().IsMonotonic())
					assert.Equal(t, pmetric.AggregationTemporalityCumulative, ms.At(i).Sum().AggregationTemporality())
					dp := ms.At(i).Sum().DataPoints().At(0)
					assert.Equal(t, start, dp.StartTimestamp())
					assert.Equal(t, ts, dp.Timestamp())
					assert.Equal(t, pmetric.NumberDataPointValueTypeInt, dp.ValueType())
					assert.Equal(t, int64(1), dp.IntValue())
					attrVal, ok := dp.Attributes().Get("app")
					assert.True(t, ok)
					assert.EqualValues(t, "pop3", attrVal.Str())
				case "m365.outlook.email_activity.count":
					assert.False(t, validatedMetrics["m365.outlook.email_activity.count"], "Found a duplicate in the metrics slice: m365.outlook.email_activity.count")
					validatedMetrics["m365.outlook.email_activity.count"] = true
					assert.Equal(t, pmetric.MetricTypeSum, ms.At(i).Type())
					assert.Equal(t, 1, ms.At(i).Sum().DataPoints().Len())
					assert.Equal(t, "The number of email actions by members over the period of time in the organization Outlook.", ms.At(i).Description())
					assert.Equal(t, "{emails}", ms.At(i).Unit())
					assert.Equal(t, false, ms.At(i).Sum().IsMonotonic())
					assert.Equal(t, pmetric.AggregationTemporalityCumulative, ms.At(i).Sum().AggregationTemporality())
					dp := ms.At(i).Sum().DataPoints().At(0)
					assert.Equal(t, start, dp.StartTimestamp())
					assert.Equal(t, ts, dp.Timestamp())
					assert.Equal(t, pmetric.NumberDataPointValueTypeInt, dp.ValueType())
					assert.Equal(t, int64(1), dp.IntValue())
					attrVal, ok := dp.Attributes().Get("activity")
					assert.True(t, ok)
					assert.EqualValues(t, "read", attrVal.Str())
				case "m365.outlook.mailboxes.active.count":
					assert.False(t, validatedMetrics["m365.outlook.mailboxes.active.count"], "Found a duplicate in the metrics slice: m365.outlook.mailboxes.active.count")
					validatedMetrics["m365.outlook.mailboxes.active.count"] = true
					assert.Equal(t, pmetric.MetricTypeSum, ms.At(i).Type())
					assert.Equal(t, 1, ms.At(i).Sum().DataPoints().Len())
					assert.Equal(t, "The number of mailboxes that have been active each day in the organization for the last 7 days.", ms.At(i).Description())
					assert.Equal(t, "{mailboxes}", ms.At(i).Unit())
					assert.Equal(t, false, ms.At(i).Sum().IsMonotonic())
					assert.Equal(t, pmetric.AggregationTemporalityCumulative, ms.At(i).Sum().AggregationTemporality())
					dp := ms.At(i).Sum().DataPoints().At(0)
					assert.Equal(t, start, dp.StartTimestamp())
					assert.Equal(t, ts, dp.Timestamp())
					assert.Equal(t, pmetric.NumberDataPointValueTypeInt, dp.ValueType())
					assert.Equal(t, int64(1), dp.IntValue())
				case "m365.outlook.quota_status.count":
					assert.False(t, validatedMetrics["m365.outlook.quota_status.count"], "Found a duplicate in the metrics slice: m365.outlook.quota_status.count")
					validatedMetrics["m365.outlook.quota_status.count"] = true
					assert.Equal(t, pmetric.MetricTypeSum, ms.At(i).Type())
					assert.Equal(t, 1, ms.At(i).Sum().DataPoints().Len())
					assert.Equal(t, "The number of mailboxes in the various quota statuses over the period of time in the org in the last 7 days.", ms.At(i).Description())
					assert.Equal(t, "{mailboxes}", ms.At(i).Unit())
					assert.Equal(t, false, ms.At(i).Sum().IsMonotonic())
					assert.Equal(t, pmetric.AggregationTemporalityCumulative, ms.At(i).Sum().AggregationTemporality())
					dp := ms.At(i).Sum().DataPoints().At(0)
					assert.Equal(t, start, dp.StartTimestamp())
					assert.Equal(t, ts, dp.Timestamp())
					assert.Equal(t, pmetric.NumberDataPointValueTypeInt, dp.ValueType())
					assert.Equal(t, int64(1), dp.IntValue())
					attrVal, ok := dp.Attributes().Get("state")
					assert.True(t, ok)
					assert.EqualValues(t, "under_limit", attrVal.Str())
				case "m365.outlook.storage.used":
					assert.False(t, validatedMetrics["m365.outlook.storage.used"], "Found a duplicate in the metrics slice: m365.outlook.storage.used")
					validatedMetrics["m365.outlook.storage.used"] = true
					assert.Equal(t, pmetric.MetricTypeSum, ms.At(i).Type())
					assert.Equal(t, 1, ms.At(i).Sum().DataPoints().Len())
					assert.Equal(t, "The amount of storage used in Outlook by the organization in the last 7 days.", ms.At(i).Description())
					assert.Equal(t, "By", ms.At(i).Unit())
					assert.Equal(t, false, ms.At(i).Sum().IsMonotonic())
					assert.Equal(t, pmetric.AggregationTemporalityCumulative, ms.At(i).Sum().AggregationTemporality())
					dp := ms.At(i).Sum().DataPoints().At(0)
					assert.Equal(t, start, dp.StartTimestamp())
					assert.Equal(t, ts, dp.Timestamp())
					assert.Equal(t, pmetric.NumberDataPointValueTypeInt, dp.ValueType())
					assert.Equal(t, int64(1), dp.IntValue())
				case "m365.sharepoint.files.active.count":
					assert.False(t, validatedMetrics["m365.sharepoint.files.active.count"], "Found a duplicate in the metrics slice: m365.sharepoint.files.active.count")
					validatedMetrics["m365.sharepoint.files.active.count"] = true
					assert.Equal(t, pmetric.MetricTypeSum, ms.At(i).Type())
					assert.Equal(t, 1, ms.At(i).Sum().DataPoints().Len())
					assert.Equal(t, "The number of active files across all sites in the last 7 days.", ms.At(i).Description())
					assert.Equal(t, "{files}", ms.At(i).Unit())
					assert.Equal(t, false, ms.At(i).Sum().IsMonotonic())
					assert.Equal(t, pmetric.AggregationTemporalityCumulative, ms.At(i).Sum().AggregationTemporality())
					dp := ms.At(i).Sum().DataPoints().At(0)
					assert.Equal(t, start, dp.StartTimestamp())
					assert.Equal(t, ts, dp.Timestamp())
					assert.Equal(t, pmetric.NumberDataPointValueTypeInt, dp.ValueType())
					assert.Equal(t, int64(1), dp.IntValue())
				case "m365.sharepoint.files.count":
					assert.False(t, validatedMetrics["m365.sharepoint.files.count"], "Found a duplicate in the metrics slice: m365.sharepoint.files.count")
					validatedMetrics["m365.sharepoint.files.count"] = true
					assert.Equal(t, pmetric.MetricTypeSum, ms.At(i).Type())
					assert.Equal(t, 1, ms.At(i).Sum().DataPoints().Len())
					assert.Equal(t, "The number of total files across all sites in the last 7 days.", ms.At(i).Description())
					assert.Equal(t, "{files}", ms.At(i).Unit())
					assert.Equal(t, false, ms.At(i).Sum().IsMonotonic())
					assert.Equal(t, pmetric.AggregationTemporalityCumulative, ms.At(i).Sum().AggregationTemporality())
					dp := ms.At(i).Sum().DataPoints().At(0)
					assert.Equal(t, start, dp.StartTimestamp())
					assert.Equal(t, ts, dp.Timestamp())
					assert.Equal(t, pmetric.NumberDataPointValueTypeInt, dp.ValueType())
					assert.Equal(t, int64(1), dp.IntValue())
				case "m365.sharepoint.pages.unique.count":
					assert.False(t, validatedMetrics["m365.sharepoint.pages.unique.count"], "Found a duplicate in the metrics slice: m365.sharepoint.pages.unique.count")
					validatedMetrics["m365.sharepoint.pages.unique.count"] = true
					assert.Equal(t, pmetric.MetricTypeSum, ms.At(i).Type())
					assert.Equal(t, 1, ms.At(i).Sum().DataPoints().Len())
					assert.Equal(t, "The number of unique views of pages across all sites in the last 7 days.", ms.At(i).Description())
					assert.Equal(t, "{views}", ms.At(i).Unit())
					assert.Equal(t, false, ms.At(i).Sum().IsMonotonic())
					assert.Equal(t, pmetric.AggregationTemporalityCumulative, ms.At(i).Sum().AggregationTemporality())
					dp := ms.At(i).Sum().DataPoints().At(0)
					assert.Equal(t, start, dp.StartTimestamp())
					assert.Equal(t, ts, dp.Timestamp())
					assert.Equal(t, pmetric.NumberDataPointValueTypeInt, dp.ValueType())
					assert.Equal(t, int64(1), dp.IntValue())
				case "m365.sharepoint.pages.viewed.count":
					assert.False(t, validatedMetrics["m365.sharepoint.pages.viewed.count"], "Found a duplicate in the metrics slice: m365.sharepoint.pages.viewed.count")
					validatedMetrics["m365.sharepoint.pages.viewed.count"] = true
					assert.Equal(t, pmetric.MetricTypeSum, ms.At(i).Type())
					assert.Equal(t, 1, ms.At(i).Sum().DataPoints().Len())
					assert.Equal(t, "The number of unique pages viewed across all sites in the last 7 days.", ms.At(i).Description())
					assert.Equal(t, "{pages}", ms.At(i).Unit())
					assert.Equal(t, false, ms.At(i).Sum().IsMonotonic())
					assert.Equal(t, pmetric.AggregationTemporalityCumulative, ms.At(i).Sum().AggregationTemporality())
					dp := ms.At(i).Sum().DataPoints().At(0)
					assert.Equal(t, start, dp.StartTimestamp())
					assert.Equal(t, ts, dp.Timestamp())
					assert.Equal(t, pmetric.NumberDataPointValueTypeInt, dp.ValueType())
					assert.Equal(t, int64(1), dp.IntValue())
				case "m365.sharepoint.site.storage.used":
					assert.False(t, validatedMetrics["m365.sharepoint.site.storage.used"], "Found a duplicate in the metrics slice: m365.sharepoint.site.storage.used")
					validatedMetrics["m365.sharepoint.site.storage.used"] = true
					assert.Equal(t, pmetric.MetricTypeSum, ms.At(i).Type())
					assert.Equal(t, 1, ms.At(i).Sum().DataPoints().Len())
					assert.Equal(t, "The amount of storage used by all sites across SharePoint in the last 7 days.", ms.At(i).Description())
					assert.Equal(t, "By", ms.At(i).Unit())
					assert.Equal(t, false, ms.At(i).Sum().IsMonotonic())
					assert.Equal(t, pmetric.AggregationTemporalityCumulative, ms.At(i).Sum().AggregationTemporality())
					dp := ms.At(i).Sum().DataPoints().At(0)
					assert.Equal(t, start, dp.StartTimestamp())
					assert.Equal(t, ts, dp.Timestamp())
					assert.Equal(t, pmetric.NumberDataPointValueTypeInt, dp.ValueType())
					assert.Equal(t, int64(1), dp.IntValue())
				case "m365.sharepoint.sites.active.count":
					assert.False(t, validatedMetrics["m365.sharepoint.sites.active.count"], "Found a duplicate in the metrics slice: m365.sharepoint.sites.active.count")
					validatedMetrics["m365.sharepoint.sites.active.count"] = true
					assert.Equal(t, pmetric.MetricTypeSum, ms.At(i).Type())
					assert.Equal(t, 1, ms.At(i).Sum().DataPoints().Len())
					assert.Equal(t, "The number of active sites across SharePoint in the last 7 days.", ms.At(i).Description())
					assert.Equal(t, "{sites}", ms.At(i).Unit())
					assert.Equal(t, false, ms.At(i).Sum().IsMonotonic())
					assert.Equal(t, pmetric.AggregationTemporalityCumulative, ms.At(i).Sum().AggregationTemporality())
					dp := ms.At(i).Sum().DataPoints().At(0)
					assert.Equal(t, start, dp.StartTimestamp())
					assert.Equal(t, ts, dp.Timestamp())
					assert.Equal(t, pmetric.NumberDataPointValueTypeInt, dp.ValueType())
					assert.Equal(t, int64(1), dp.IntValue())
				case "m365.teams.calls.count":
					assert.False(t, validatedMetrics["m365.teams.calls.count"], "Found a duplicate in the metrics slice: m365.teams.calls.count")
					validatedMetrics["m365.teams.calls.count"] = true
					assert.Equal(t, pmetric.MetricTypeSum, ms.At(i).Type())
					assert.Equal(t, 1, ms.At(i).Sum().DataPoints().Len())
					assert.Equal(t, "The number of MS Teams calls from users in the organization in the last 7 days.", ms.At(i).Description())
					assert.Equal(t, "{calls}", ms.At(i).Unit())
					assert.Equal(t, false, ms.At(i).Sum().IsMonotonic())
					assert.Equal(t, pmetric.AggregationTemporalityCumulative, ms.At(i).Sum().AggregationTemporality())
					dp := ms.At(i).Sum().DataPoints().At(0)
					assert.Equal(t, start, dp.StartTimestamp())
					assert.Equal(t, ts, dp.Timestamp())
					assert.Equal(t, pmetric.NumberDataPointValueTypeInt, dp.ValueType())
					assert.Equal(t, int64(1), dp.IntValue())
				case "m365.teams.device_usage.users":
					assert.False(t, validatedMetrics["m365.teams.device_usage.users"], "Found a duplicate in the metrics slice: m365.teams.device_usage.users")
					validatedMetrics["m365.teams.device_usage.users"] = true
					assert.Equal(t, pmetric.MetricTypeSum, ms.At(i).Type())
					assert.Equal(t, 1, ms.At(i).Sum().DataPoints().Len())
					assert.Equal(t, "The number of unique users by device/platform that have used Teams in the last 7 days.", ms.At(i).Description())
					assert.Equal(t, "{users}", ms.At(i).Unit())
					assert.Equal(t, false, ms.At(i).Sum().IsMonotonic())
					assert.Equal(t, pmetric.AggregationTemporalityCumulative, ms.At(i).Sum().AggregationTemporality())
					dp := ms.At(i).Sum().DataPoints().At(0)
					assert.Equal(t, start, dp.StartTimestamp())
					assert.Equal(t, ts, dp.Timestamp())
					assert.Equal(t, pmetric.NumberDataPointValueTypeInt, dp.ValueType())
					assert.Equal(t, int64(1), dp.IntValue())
					attrVal, ok := dp.Attributes().Get("device")
					assert.True(t, ok)
					assert.EqualValues(t, "Android", attrVal.Str())
				case "m365.teams.meetings.count":
					assert.False(t, validatedMetrics["m365.teams.meetings.count"], "Found a duplicate in the metrics slice: m365.teams.meetings.count")
					validatedMetrics["m365.teams.meetings.count"] = true
					assert.Equal(t, pmetric.MetricTypeSum, ms.At(i).Type())
					assert.Equal(t, 1, ms.At(i).Sum().DataPoints().Len())
					assert.Equal(t, "The number of MS Teams meetings for users in the organization in the last 7 days.", ms.At(i).Description())
					assert.Equal(t, "{meetings}", ms.At(i).Unit())
					assert.Equal(t, false, ms.At(i).Sum().IsMonotonic())
					assert.Equal(t, pmetric.AggregationTemporalityCumulative, ms.At(i).Sum().AggregationTemporality())
					dp := ms.At(i).Sum().DataPoints().At(0)
					assert.Equal(t, start, dp.StartTimestamp())
					assert.Equal(t, ts, dp.Timestamp())
					assert.Equal(t, pmetric.NumberDataPointValueTypeInt, dp.ValueType())
					assert.Equal(t, int64(1), dp.IntValue())
				case "m365.teams.messages.private.count":
					assert.False(t, validatedMetrics["m365.teams.messages.private.count"], "Found a duplicate in the metrics slice: m365.teams.messages.private.count")
					validatedMetrics["m365.teams.messages.private.count"] = true
					assert.Equal(t, pmetric.MetricTypeSum, ms.At(i).Type())
					assert.Equal(t, 1, ms.At(i).Sum().DataPoints().Len())
					assert.Equal(t, "The number of MS Teams private-messages sent by users in the organization in the last 7 days.", ms.At(i).Description())
					assert.Equal(t, "{messages}", ms.At(i).Unit())
					assert.Equal(t, false, ms.At(i).Sum().IsMonotonic())
					assert.Equal(t, pmetric.AggregationTemporalityCumulative, ms.At(i).Sum().AggregationTemporality())
					dp := ms.At(i).Sum().DataPoints().At(0)
					assert.Equal(t, start, dp.StartTimestamp())
					assert.Equal(t, ts, dp.Timestamp())
					assert.Equal(t, pmetric.NumberDataPointValueTypeInt, dp.ValueType())
					assert.Equal(t, int64(1), dp.IntValue())
				case "m365.teams.messages.team.count":
					assert.False(t, validatedMetrics["m365.teams.messages.team.count"], "Found a duplicate in the metrics slice: m365.teams.messages.team.count")
					validatedMetrics["m365.teams.messages.team.count"] = true
					assert.Equal(t, pmetric.MetricTypeSum, ms.At(i).Type())
					assert.Equal(t, 1, ms.At(i).Sum().DataPoints().Len())
					assert.Equal(t, "The number of MS Teams team-messages sent by users in the organization in the last 7 days.", ms.At(i).Description())
					assert.Equal(t, "{messages}", ms.At(i).Unit())
					assert.Equal(t, false, ms.At(i).Sum().IsMonotonic())
					assert.Equal(t, pmetric.AggregationTemporalityCumulative, ms.At(i).Sum().AggregationTemporality())
					dp := ms.At(i).Sum().DataPoints().At(0)
					assert.Equal(t, start, dp.StartTimestamp())
					assert.Equal(t, ts, dp.Timestamp())
					assert.Equal(t, pmetric.NumberDataPointValueTypeInt, dp.ValueType())
					assert.Equal(t, int64(1), dp.IntValue())
				}
			}
		})
	}
}
