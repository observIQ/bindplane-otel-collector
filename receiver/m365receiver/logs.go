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

package m365receiver // import "github.com/observiq/observiq-otel-collector/receiver/m365receiver"

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/stanza/adapter"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/extension/experimental/storage"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/receiver"
	"go.uber.org/zap"
)

const (
	logStorageKey = "last_recorded_log"
	layout        = "2006-01-02T15:04:05"
)

type lClient interface {
	GetJSON(ctx context.Context, endpoint string, end string, start string) ([]logData, error)
	GetToken(ctx context.Context) error
	StartSubscription(ctx context.Context, endpoint string) error
	shutdown() error
}

type m365LogsReceiver struct {
	settings      component.TelemetrySettings
	logger        *zap.Logger
	consumer      consumer.Logs
	cfg           *Config
	client        lClient
	storageClient storage.Client
	id            component.ID

	wg           *sync.WaitGroup
	mu           sync.Mutex
	pollInterval time.Duration
	cancel       context.CancelFunc
	audits       []auditMetaData
	record       *logRecord
	root         string
	startRoot    string
}

type logRecord struct {
	NextStartTime *time.Time `mapstructure:"next_start_time"`
}

type auditMetaData struct {
	name    string
	route   string
	enabled bool
}

func newM365Logs(cfg *Config, settings receiver.CreateSettings, consumer consumer.Logs) *m365LogsReceiver {
	return &m365LogsReceiver{
		settings:      settings.TelemetrySettings,
		logger:        settings.Logger,
		consumer:      consumer,
		cfg:           cfg,
		storageClient: storage.NewNopClient(),
		id:            settings.ID,
		wg:            &sync.WaitGroup{},
		pollInterval:  cfg.Logs.PollInterval,
		audits: []auditMetaData{
			{"general", "Audit.General", cfg.Logs.GeneralLogs},
			{"exchange", "Audit.Exchange", cfg.Logs.ExchangeLogs},
			{"sharepoint", "Audit.SharePoint", cfg.Logs.SharepointLogs},
			{"azureAD", "Audit.AzureActiveDirectory", cfg.Logs.AzureADLogs},
			{"dlp", "DLP.All", cfg.Logs.DLPLogs},
		},
		root:      fmt.Sprintf("https://manage.office.com/api/v1.0/%s/activity/feed/subscriptions/content?contentType=", cfg.TenantID),
		startRoot: fmt.Sprintf("https://manage.office.com/api/v1.0/%s/activity/feed/subscriptions/start?contentType=", cfg.TenantID),
	}
}

// creates the client for http requests, and initializes metadata struct
func (l *m365LogsReceiver) Start(ctx context.Context, host component.Host) error {
	// create default client
	httpClient, err := l.cfg.ToClient(host, l.settings)
	if err != nil {
		l.logger.Error("error creating HTTP client", zap.Error(err))
		return err
	}

	// create m365 log client, create token and start audit subscriptions
	l.client = newM365Client(httpClient, l.cfg, "https://manage.office.com/.default")
	err = l.client.GetToken(ctx)
	if err != nil {
		l.logger.Error("error creating authorization token", zap.Error(err))
		return err
	}
	for _, a := range l.audits {
		err = l.client.StartSubscription(ctx, l.startRoot+a.route)
		if err != nil {
			l.logger.Error("error starting audit subscriptions", zap.Error(err))
			return err
		}
	}

	// set cancel function
	cancelCtx, cancel := context.WithCancel(ctx)
	l.cancel = cancel

	// init checkpoint stuff
	storageClient, err := adapter.GetStorageClient(ctx, host, l.cfg.StorageID, l.id)
	if err != nil {
		return fmt.Errorf("failed to get storage client: %w", err)
	}
	l.storageClient = storageClient
	l.loadCheckpoint(cancelCtx)

	return l.startPolling(cancelCtx)
}

func (l *m365LogsReceiver) Shutdown(ctx context.Context) error {
	l.logger.Debug("shutting down logs receiver")
	l.cancel()
	l.wg.Wait()
	return l.checkpoint(ctx)
}

// spins a go routine at each poll interval to go get logs
func (l *m365LogsReceiver) startPolling(ctx context.Context) error {
	t := time.NewTicker(l.pollInterval)
	l.wg.Add(1)
	go func() {
		defer l.wg.Done()
		defer t.Stop()
		for {
			select {
			case <-t.C:
				if err := l.pollLogs(ctx); err != nil {
					l.logger.Error("error while polling for logs", zap.Error(err))
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	return nil
}

// spins a go routine for each audit type/endpoint
func (l *m365LogsReceiver) pollLogs(ctx context.Context) error {
	var st string
	if l.record.NextStartTime != nil {
		st = l.record.NextStartTime.Format(layout)
	} else {
		st = pcommon.NewTimestampFromTime(time.Now().Add(-l.pollInterval)).AsTime().UTC().Format(layout)
	}

	now := time.Now().UTC()

	auditWG := &sync.WaitGroup{}
	for i := 0; i < len(l.audits); i++ {
		auditWG.Add(1)
		go l.poll(ctx, now, st, &l.audits[i], auditWG)
	}
	auditWG.Wait()

	l.record.NextStartTime = &now
	return l.checkpoint(ctx)
}

// collects log data from endpoint, transforms logs, consumes logs
func (l *m365LogsReceiver) poll(ctx context.Context, now time.Time, st string, audit *auditMetaData, wg *sync.WaitGroup) {
	defer wg.Done()
	if !audit.enabled {
		return
	}

	data, err := l.getLogs(ctx, now.Format(layout), st, audit)
	if err != nil {
		return
	}

	logs := l.transformLogs(pcommon.NewTimestampFromTime(now), audit, data)

	if logs.LogRecordCount() > 0 {
		if err = l.consumer.ConsumeLogs(ctx, logs); err != nil {
			l.logger.Error("error consuming events", zap.Error(err))
		}
	}
}

func (l *m365LogsReceiver) getLogs(ctx context.Context, end string, start string, audit *auditMetaData) ([]logData, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	data, err := l.client.GetJSON(ctx, l.root+audit.route, end, start)
	if err != nil {
		if err.Error() == "authorization denied" { // troubleshoot stale token
			l.logger.Debug("possible stale token; attempting to regenerate")
			err = l.client.GetToken(ctx)
			if err != nil { // something went wrong generating token
				l.logger.Error("error creating authorization token", zap.Error(err))
				return []logData{}, err
			}
			data, err = l.client.GetJSON(ctx, l.root+audit.route, end, start)
			if err != nil { // not a stale token error, unsure what is wrong
				l.logger.Error("unable to retrieve logs", zap.Error(err))
				return []logData{}, err
			}
		} else {
			l.logger.Error("error retrieving logs", zap.Error(err))
			return []logData{}, err
		}
	}

	return data, nil
}

// constructs logs from logData
func (l *m365LogsReceiver) transformLogs(now pcommon.Timestamp, audit *auditMetaData, data []logData) plog.Logs {
	logs := plog.NewLogs()
	resourceLogs := logs.ResourceLogs().AppendEmpty()
	scopeLogs := resourceLogs.ScopeLogs().AppendEmpty()

	ra := resourceLogs.Resource().Attributes()
	ra.PutStr("m365.audit", audit.name)
	ra.PutStr("m365.organization.id", l.cfg.TenantID)

	for _, log := range data {
		logRecord := scopeLogs.LogRecords().AppendEmpty()

		// log body
		logRecord.Body().SetStr(log.body)

		// timestamp
		ts, err := time.Parse(layout, log.log.CreationTime)
		if err != nil {
			l.logger.Warn("unable to interpret when an event was created, expecting a RFC3339 timestamp", zap.String("timestamp", log.log.CreationTime), zap.String("log", log.log.ID))
			logRecord.SetTimestamp(now)
		} else {
			logRecord.SetTimestamp(pcommon.NewTimestampFromTime(ts))
		}
		logRecord.SetObservedTimestamp(now)

		// attributes
		attrs := logRecord.Attributes()
		attrs.PutStr("id", log.log.ID)
		attrs.PutStr("operation", log.log.Operation)
		attrs.PutStr("user.id", log.log.UserID)
		attrs.PutStr("user.type", matchUserType(log.log.UserType))

		// optional attributes
		parseOptionalAttributes(&attrs, &log.log)
	}

	return logs
}

// sets the checkpoint
func (l *m365LogsReceiver) checkpoint(ctx context.Context) error {
	bytes, err := json.Marshal(l.record)
	if err != nil {
		return fmt.Errorf("unable to write checkpoint: %w", err)
	}
	return l.storageClient.Set(ctx, logStorageKey, bytes)
}

// loads the checkpoint
func (l *m365LogsReceiver) loadCheckpoint(ctx context.Context) {
	bytes, err := l.storageClient.Get(ctx, logStorageKey)
	if err != nil {
		l.logger.Info("unable to load checkpoint from storage client, continuing without a previous checkpoint", zap.Error(err))
		l.record = &logRecord{}
		return
	}

	if bytes == nil {
		l.record = &logRecord{}
		return
	}

	var record logRecord
	if err = json.Unmarshal(bytes, &record); err != nil {
		l.logger.Error("unable to decode stored record for logs, continuing without a checkpoint", zap.Error(err))
		l.record = &logRecord{}
		return
	}
	l.record = &record
}

// for jsonLog fields that are strings
func setAttributeIfNotEmpty(m *pcommon.Map, field, value string) {
	if value != "" {
		m.PutStr(field, value)
	}
}

// adds any attributes to log that may or may not be present
func parseOptionalAttributes(m *pcommon.Map, log *jsonLog) {
	setAttributeIfNotEmpty(m, "workload", log.Workload)
	setAttributeIfNotEmpty(m, "result", log.ResultStatus)
	setAttributeIfNotEmpty(m, "sharepoint.site.id", log.SharepointSite)
	setAttributeIfNotEmpty(m, "exchange.mailbox.id", log.ExchangeMailboxGUID)
	setAttributeIfNotEmpty(m, "yammer.user.id", log.YammerActorID)
	setAttributeIfNotEmpty(m, "investigation.id", log.InvestigationID)
	setAttributeIfNotEmpty(m, "investigation.status", log.InvestigationStatus)
	setAttributeIfNotEmpty(m, "dynamics365.entity.id", log.DynamicsEntityID)
	setAttributeIfNotEmpty(m, "dynamics365.entity.name", log.DynamicsEntityName)
	setAttributeIfNotEmpty(m, "forms.form.id", log.FormID)
	setAttributeIfNotEmpty(m, "mip.label.id", log.MIPLabelID)
	setAttributeIfNotEmpty(m, "todo.app.id", log.MSToDoAppID)
	setAttributeIfNotEmpty(m, "todo.item.id", log.MSToDoItemID)
	setAttributeIfNotEmpty(m, "web.project.id", log.MSWebProjectID)
	setAttributeIfNotEmpty(m, "web.roadmap.id", log.MSWebRoadmapID)
	setAttributeIfNotEmpty(m, "web.roadmap.item.id", log.MSWebRoadmapItemID)

	if log.YammerFileID != nil {
		m.PutStr("yammer.file.id", strconv.Itoa(*log.YammerFileID))
	}
	if log.DLPSharePointMetaData != nil {
		m.PutStr("dlp.sharepoint.user", log.DLPSharePointMetaData.From)
	}
	if log.DLPExchangeMetaData != nil {
		m.PutStr("dlp.exchange.message.id", log.DLPExchangeMetaData.MessageID)
	}
	if log.QuarantineSource != nil {
		m.PutStr("quarantine.request_source", matchQuarantineSource(*log.QuarantineSource))
	}
	if log.DefenderFileSource != nil {
		m.PutStr("defender.file.source_workload", matchSourceWorkload(*log.DefenderFileSource))
	}
	if log.CommCompliance != nil {
		m.PutStr("communication_compliance.exchange.network.message.id", log.CommCompliance.NetworkMessageID)
	}
	if log.DataShareInvitation != nil {
		m.PutStr("system_sync.data.share.id", log.DataShareInvitation.ShareID)
	}
	if log.DefenderFile != nil {
		m.PutStr("defender.file.id", log.DefenderFile.DocumentID)
		m.PutStr("defender.file.verdict", matchFileVerdict(log.DefenderFile.FileVerdict))
	}
	if log.SecurityAlertID != "" {
		m.PutStr("security.alert.id", log.SecurityAlertID)
		m.PutStr("security.alert.name", log.SecurityAlertName)
	}
	if log.ConnectorJobID != "" {
		m.PutStr("compliance_connector.job.id", log.ConnectorJobID)
		setAttributeIfNotEmpty(m, "compliance_connector.task.id", log.ConnectorTaskID)
	}
	if log.VivaGoalsUsername != "" {
		m.PutStr("viva.username", log.VivaGoalsUsername)
		setAttributeIfNotEmpty(m, "viva.organization.name", log.VivaGoalsOrgName)
	}

	recordType := matchRecordType(log.RecordType)
	if recordType == "ThreatIntelligenceUrl" {
		m.PutStr("defender.url.url", log.DefenderURL)
	}
	if recordType == "PowerBIAudit" {
		m.PutStr("powerbi.app.name", log.PowerAppName)
	}
	if recordType == "OMEPortal" {
		setAttributeIfNotEmpty(m, "encrypted_portal.message.id", log.EncryptedMessageID)
	}
	if recordType == "MicrosoftGraphDataConnectConsent" {
		setAttributeIfNotEmpty(m, "graph.consent.app.id", log.MSGraphConsentAppID)
	}

	if log.AzureActor != nil {
		slice := m.PutEmptySlice("azure.actors")
		for _, r := range *log.AzureActor {
			pair := slice.AppendEmpty().SetEmptyMap()
			pair.PutStr("actor.id", r.ID)
			pair.PutStr("actor.type", matchAzureUserType(r.Type))
		}
	}
	if log.DLPPolicyDetails != nil {
		slice := m.PutEmptySlice("dlp.policy_details")
		for _, r := range *log.DLPPolicyDetails {
			pair := slice.AppendEmpty().SetEmptyMap()
			pair.PutStr("policy.id", r.PolicyID)
			pair.PutStr("policy.name", r.PolicyName)
		}
	}

	if log.DefenderEmail != nil {
		slice := m.PutEmptySlice("defender.email.attachments")
		for _, r := range *log.DefenderEmail {
			pair := slice.AppendEmpty().SetEmptyMap()
			pair.PutStr("file.name", r.FileName)
		}
	}

}

func matchUserType(x int) string {
	switch x {
	case 0:
		return "Regular"
	case 1:
		return "Reserved"
	case 2:
		return "Admin"
	case 3:
		return "DcAdmin"
	case 4:
		return "System"
	case 5:
		return "Application"
	case 6:
		return "ServicePrincipal"
	case 7:
		return "CustomPolicy"
	case 8:
		return "SystemPolicy"
	default:
		return "unknown"
	}
}

func matchAzureUserType(x int) string {
	switch x {
	case 0:
		return "Claim"
	case 1:
		return "Name"
	case 2:
		return "Other"
	case 3:
		return "PUID"
	case 4:
		return "SPN"
	case 5:
		return "UPN"
	default:
		return "unknown"
	}
}

func matchFileVerdict(x int) string {
	switch x {
	case 0:
		return "Good"
	case 1:
		return "Bad"
	case -1:
		return "Error"
	case -2:
		return "Timeout"
	case -3:
		return "Pending"
	default:
		return "unknown"
	}
}

func matchSourceWorkload(x int) string {
	switch x {
	case 0:
		return "SharePoint Online"
	case 1:
		return "OneDrive for Business"
	case 2:
		return "Microsoft Teams"
	default:
		return "unknown"
	}
}

func matchQuarantineSource(x int) string {
	switch x {
	case 0:
		return "SCC"
	case 1:
		return "Cmdlet"
	case 2:
		return "URLlink"
	default:
		return "unknown"
	}
}
func matchRecordType(x int) string {
	switch x {
	case 20:
		return "PowerBIAudit"
	case 41:
		return "ThreatIntelligenceUrl"
	case 154:
		return "OMEPortal"
	case 217:
		return "MicrosoftGraphDataConnectConsent"
	default:
		return "ignore"
	}
}
