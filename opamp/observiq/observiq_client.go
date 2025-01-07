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

// Package observiq contains OpAmp structures compatible with the observiq client
package observiq

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"sync"

	"github.com/observiq/bindplane-otel-collector/collector"
	"github.com/observiq/bindplane-otel-collector/internal/measurements"
	"github.com/observiq/bindplane-otel-collector/internal/report"
	"github.com/observiq/bindplane-otel-collector/internal/topology"
	"github.com/observiq/bindplane-otel-collector/internal/version"
	"github.com/observiq/bindplane-otel-collector/opamp"
	"github.com/observiq/bindplane-otel-collector/packagestate"
	"github.com/open-telemetry/opamp-go/client"
	"github.com/open-telemetry/opamp-go/client/types"
	"github.com/open-telemetry/opamp-go/protobufs"
	"go.uber.org/zap"
)

var (
	// ErrUnsupportedURL is error returned when creating a client with an unsupported URL scheme
	ErrUnsupportedURL = errors.New("unsupported URL")
)

const capabilities = protobufs.AgentCapabilities_AgentCapabilities_ReportsStatus |
	protobufs.AgentCapabilities_AgentCapabilities_AcceptsPackages |
	protobufs.AgentCapabilities_AgentCapabilities_ReportsPackageStatuses |
	protobufs.AgentCapabilities_AgentCapabilities_ReportsEffectiveConfig |
	protobufs.AgentCapabilities_AgentCapabilities_AcceptsRemoteConfig |
	protobufs.AgentCapabilities_AgentCapabilities_ReportsRemoteConfig

// Ensure interface is satisfied
var _ opamp.Client = (*Client)(nil)

// Client represents a client that is connected to Iris via OpAmp
type Client struct {
	opampClient             client.OpAMPClient
	logger                  *zap.Logger
	ident                   *identity
	configManager           opamp.ConfigManager
	downloadableFileManager opamp.DownloadableFileManager
	collector               collector.Collector
	packagesStateProvider   types.PackagesStateProvider
	updaterManager          updaterManager
	mutex                   sync.Mutex
	updatingPackage         bool
	reportManager           *report.Manager
	measurementsSender      *measurementsSender
	topologySender          *topologySender

	// To signal if we are disconnecting already and not take any actions on connection failures
	disconnecting bool

	// Used to monitor collector status
	collectorMntrCtx    context.Context
	collectorMntrCancel context.CancelFunc
	collectorMntrWg     sync.WaitGroup

	currentConfig opamp.Config
}

// NewClientArgs arguments passed when creating a new client
type NewClientArgs struct {
	DefaultLogger *zap.Logger
	Config        opamp.Config
	Collector     collector.Collector
	Version       string

	TmpPath              string
	ManagerConfigPath    string
	CollectorConfigPath  string
	LoggerConfigPath     string
	MeasurementsReporter MeasurementsReporter
	TopologyReporter     TopologyReporter
}

// NewClient creates a new OpAmp client
func NewClient(args *NewClientArgs) (opamp.Client, error) {
	clientLogger := args.DefaultLogger.Named("opamp")

	configManager := NewAgentConfigManager(args.DefaultLogger)
	updaterManger, err := newUpdaterManager(clientLogger, args.TmpPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create updaterManager: %w", err)
	}

	// Propagate TLS config to reportManager agent
	tlsCfg, err := args.Config.ToTLS(nil)
	if err != nil {
		return nil, fmt.Errorf("failed creating TLS config: %w", err)
	}

	reportManager := report.GetManager()
	if err := reportManager.SetClient(report.NewAgentClient(args.Config.AgentID.String(), args.Config.SecretKey, tlsCfg)); err != nil {
		// Error should never happen as we only error if a nil client is sent
		return nil, fmt.Errorf("failed to set client on report manager: %w", err)
	}

	observiqClient := &Client{
		logger:                  clientLogger,
		ident:                   newIdentity(clientLogger, args.Config, args.Version),
		configManager:           configManager,
		downloadableFileManager: newDownloadableFileManager(clientLogger, args.TmpPath),
		collector:               args.Collector,
		currentConfig:           args.Config,
		packagesStateProvider:   newPackagesStateProvider(clientLogger, packagestate.DefaultFileName),
		updaterManager:          updaterManger,
		reportManager:           reportManager,
	}

	// Parse URL to determin scheme
	opampURL, err := url.Parse(args.Config.Endpoint)
	if err != nil {
		return nil, err
	}

	// Add managed configs
	if err := observiqClient.addManagedConfigs(args); err != nil {
		return nil, err
	}

	// Create collect client based on URL scheme
	switch opampURL.Scheme {
	case "ws", "wss":
		logger := newZapOpAMPLoggerAdapter(clientLogger)
		observiqClient.opampClient = client.NewWebSocket(logger)
	default:
		return nil, ErrUnsupportedURL
	}

	err = observiqClient.opampClient.SetCustomCapabilities(&protobufs.CustomCapabilities{
		Capabilities: []string{
			measurements.ReportMeasurementsV1Capability,
			topology.ReportTopologyCapability,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("error setting custom capabilities: %w", err)
	}

	// Create measurements sender
	observiqClient.measurementsSender = newMeasurementsSender(
		clientLogger,
		args.MeasurementsReporter,
		observiqClient.opampClient,
		args.Config.MeasurementsInterval,
		args.Config.ExtraMeasurementsAttributes,
	)

	// Create topology sender
	observiqClient.topologySender = newTopologySender(
		clientLogger,
		args.TopologyReporter,
		observiqClient.opampClient,
	)

	return observiqClient, nil
}

func (c *Client) addManagedConfigs(args *NewClientArgs) error {
	// Add configs to config manager
	managerManagedConfig, err := opamp.NewManagedConfig(args.ManagerConfigPath, managerReload(c, args.ManagerConfigPath), true)
	if err != nil {
		return fmt.Errorf("failed to create manager managed config: %w", err)
	}
	c.configManager.AddConfig(ManagerConfigName, managerManagedConfig)

	collectorManagedConfig, err := opamp.NewManagedConfig(args.CollectorConfigPath, collectorReload(c, args.CollectorConfigPath), true)
	if err != nil {
		return fmt.Errorf("failed to create collector managed config: %w", err)
	}
	c.configManager.AddConfig(CollectorConfigName, collectorManagedConfig)

	loggerManagedConfig, err := opamp.NewManagedConfig(args.LoggerConfigPath, loggerReload(c, args.LoggerConfigPath), true)
	if err != nil {
		return fmt.Errorf("failed to create logger managed config: %w", err)
	}
	c.configManager.AddConfig(LoggingConfigName, loggerManagedConfig)

	reportManagedConfig, err := opamp.NewManagedConfig("report.yaml", reportReload(c), false)
	if err != nil {
		return fmt.Errorf("failed to create report managed config: %w", err)
	}
	c.configManager.AddConfig(ReportConfigName, reportManagedConfig)

	return nil
}

// Connect initiates a connection to the OpAmp server
func (c *Client) Connect(ctx context.Context) error {
	// Compose and set the agent description
	if err := c.opampClient.SetAgentDescription(c.ident.ToAgentDescription()); err != nil {
		c.logger.Error("Error while setting agent description", zap.Error(err))

		// Set package status file for error (for Updater to pick up), but do not force send to Server
		c.tryToFailPackageInstall(fmt.Sprintf("Failed setting agent description: %s", err.Error()), false)

		return err
	}

	tlsCfg, err := c.currentConfig.ToTLS(nil)
	if err != nil {
		// Set package status file for error (for Updater to pick up), but do not force send to Server
		c.tryToFailPackageInstall(fmt.Sprintf("Failed creating TLS config: %s", err.Error()), false)

		return fmt.Errorf("failed creating TLS config: %w", err)
	}

	settings := types.StartSettings{
		OpAMPServerURL: c.currentConfig.Endpoint,
		Header: http.Header{
			"Authorization":               []string{fmt.Sprintf("Secret-Key %s", c.currentConfig.GetSecretKey())},
			"User-Agent":                  []string{fmt.Sprintf("observiq-otel-collector/%s", version.Version())},
			"OpAMP-Version":               []string{opamp.Version()},
			"Agent-ID":                    []string{c.ident.agentID.String()},
			"Agent-Version":               []string{version.Version()},
			"Agent-Hostname":              []string{c.ident.hostname},
			"X-Bindplane-Agent-Id-Format": []string{c.ident.agentID.Type()},
		},
		TLSConfig:   tlsCfg,
		InstanceUid: c.ident.agentID.OpAMPInstanceUID(),
		Callbacks: types.CallbacksStruct{
			OnConnectFunc:          c.onConnectHandler,
			OnConnectFailedFunc:    c.onConnectFailedHandler,
			OnErrorFunc:            c.onErrorHandler,
			OnMessageFunc:          c.onMessageFuncHandler,
			GetEffectiveConfigFunc: c.onGetEffectiveConfigHandler,
			// Unimplemented handlers
			// OnOpampConnectionSettingsFunc
			// OnOpampConnectionSettingsAcceptedFunc
			// OnCommandFunc
			// SaveRemoteConfigStatusFunc
		},
		PackagesStateProvider: c.packagesStateProvider,
		Capabilities:          capabilities,
	}

	// Start the embedded collector
	// Pass in the background context here so it's clear we need to shutdown the collector instead
	// of the context shutting it down via a cancel.
	if err := c.collector.Run(context.Background()); err != nil {
		// Set package status file for error (for Updater to pick up), but do not force send to Server
		c.tryToFailPackageInstall(fmt.Sprintf("Collector failed to start: %s", err.Error()), false)

		return fmt.Errorf("collector failed to start: %w", err)
	}

	// Now that collector has successfully started kick off monitoring
	c.startCollectorMonitoring(ctx)

	err = c.opampClient.Start(ctx, settings)
	if err != nil {
		// Set package status file for error (for Updater to pick up), but do not force send to Server
		c.tryToFailPackageInstall(fmt.Sprintf("OpAMP client failed to start: %s", err.Error()), false)
	}

	return err
}

// Disconnect disconnects from the server
func (c *Client) Disconnect(ctx context.Context) error {
	// Ensure we're no longer monitoring the collector as we shutdown to avoid error messages due to shutdown
	c.stopCollectorMonitoring()

	c.safeSetDisconnecting(true)
	c.collector.Stop(ctx)
	return c.opampClient.Stop(ctx)
}

// client callbacks

func (c *Client) onConnectHandler(_ context.Context) {
	c.logger.Info("Successfully connected to server")

	// See if we can retrieve the PackageStatuses where the collector package is in an installing state
	pkgStatuses, err := c.getVerifiedPackageStatuses()
	if err != nil {
		c.logger.Error("Problem with PackageStatuses", zap.Error(err))
		return
	}

	collectorPkgStatus := pkgStatuses.Packages[packagestate.CollectorPackageName]
	// If we were not installing before the connection, nothing else to do
	if collectorPkgStatus.Status != protobufs.PackageStatusEnum_PackageStatusEnum_Installing {
		return
	}

	// If in the middle of an install and we just connected, this is most likely becasue the collector was just spun up fresh by the Updater.
	// If the current version matches the server offered version, this implies a good install and so we should set the PackageStatuses and
	// send it to the OpAMP Server. If the version does not match, just change the PackageStatues JSON so that the Updater can start rollback.

	if collectorPkgStatus.ServerOfferedVersion != version.Version() {
		errMsg := fmt.Sprintf("Failed because of collector version mismatch: expected %s, actual %s",
			collectorPkgStatus.ServerOfferedVersion, version.Version())
		c.failPackageInstall(pkgStatuses, errMsg, false)

		return
	}

	// Installation of new collector was successful!
	c.finishPackageInstall(pkgStatuses)
}

func (c *Client) onConnectFailedHandler(_ context.Context, err error) {
	c.logger.Error("Failed to connect to server", zap.Error(err))

	// We are currently disconnecting so any Connection failed error is expected and should not affect an install
	if !c.safeGetDisconnecting() {
		// Set package status file for error (for Updater to pick up), but do not force send to Server
		c.tryToFailPackageInstall(fmt.Sprintf("Failed to connect to OpAMP Server: %s", err.Error()), false)
	}
}

func (c *Client) onErrorHandler(_ context.Context, errResp *protobufs.ServerErrorResponse) {
	c.logger.Error("Server returned an error response", zap.String("Error", errResp.GetErrorMessage()))
}

func (c *Client) onGetEffectiveConfigHandler(_ context.Context) (*protobufs.EffectiveConfig, error) {
	c.logger.Debug("Remote Compose Effective config handler")
	return c.configManager.ComposeEffectiveConfig()
}

func (c *Client) onMessageFuncHandler(ctx context.Context, msg *types.MessageData) {
	c.logger.Debug("On message handler")
	if msg.RemoteConfig != nil {
		if err := c.onRemoteConfigHandler(ctx, msg.RemoteConfig); err != nil {
			c.logger.Error("Error while processing Remote Config Change", zap.Error(err))
		}
	}
	if msg.PackagesAvailable != nil {
		if err := c.onPackagesAvailableHandler(msg.PackagesAvailable); err != nil {
			c.logger.Error("Error while processing Packages Available Change", zap.Error(err))
		}
	}
	if msg.CustomCapabilities != nil {
		if slices.Contains(msg.CustomCapabilities.Capabilities, measurements.ReportMeasurementsV1Capability) {
			c.logger.Info("Server supports custom throughput message measurements, starting measurements sender.")
			c.measurementsSender.Start()
		} else {
			c.logger.Info("Server does not support custom throughput message measurements, stopping measurements sender.")
			c.measurementsSender.Stop()
		}
		if slices.Contains(msg.CustomCapabilities.Capabilities, topology.ReportTopologyCapability) {
			c.logger.Info("Server supports custom topology messages, starting topology sender.")
			c.topologySender.Start()
		} else {
			c.logger.Info("Server does not support custom topology messages, stopping topology sender.")
			c.topologySender.Stop()
		}
	}
}

func (c *Client) onRemoteConfigHandler(ctx context.Context, remoteConfig *protobufs.AgentRemoteConfig) error {
	c.logger.Debug("Remote config handler")

	changed, err := c.configManager.ApplyConfigChanges(remoteConfig)
	remoteCfgStatus := &protobufs.RemoteConfigStatus{
		LastRemoteConfigHash: remoteConfig.GetConfigHash(),
		Status:               protobufs.RemoteConfigStatuses_RemoteConfigStatuses_APPLIED,
	}

	// If we received and error apply it to the config
	if err != nil {
		c.logger.Error("Failed applying remote config", zap.Error(err))

		remoteCfgStatus.Status = protobufs.RemoteConfigStatuses_RemoteConfigStatuses_FAILED
		remoteCfgStatus.ErrorMessage = fmt.Sprintf("Failed to apply config changes: %s", err.Error())
	}

	c.logger.Info("Setting remote config status", zap.String("status", remoteCfgStatus.Status.String()))
	// Set the remote config status
	if err := c.opampClient.SetRemoteConfigStatus(remoteCfgStatus); err != nil {
		return fmt.Errorf("failed to set remote config status: %w", err)
	}

	// If we changed the config call UpdateEffectiveConfig
	if changed {
		c.logger.Info("Updating effective config")
		if err := c.opampClient.UpdateEffectiveConfig(ctx); err != nil {
			return fmt.Errorf("failed to update effective config: %w", err)
		}
	}
	return nil
}

// onPackagesAvailableHandler handles when OnMessage contains a PackagesAvailable message
func (c *Client) onPackagesAvailableHandler(availablePkgs *protobufs.PackagesAvailable) error {
	c.logger.Debug("Packages available handler")

	// Initialize PackageStatuses that will eventually be sent back to server
	curPkgStatuses := &protobufs.PackageStatuses{
		ServerProvidedAllPackagesHash: availablePkgs.GetAllPackagesHash(),
		Packages:                      map[string]*protobufs.PackageStatus{},
	}

	// Don't respond to PackagesAvailable messages while currently installing. We use this in memory data rather than the
	// PackageStatuses persistant data in order to ensure that we don't get stuck in a stuck state
	if c.safeGetUpdatingPackage() {
		c.logger.Warn(
			"Not starting new package update as already installing new packages",
			zap.String("AllPackagesHash", hex.EncodeToString(availablePkgs.GetAllPackagesHash())))

		curPkgStatuses.ErrorMessage = "Already installing new packages"
		// Dont' actually set the on file package statuses because we want to ignore this
		if err := c.opampClient.SetPackageStatuses(curPkgStatuses); err != nil {
			c.logger.Error("OpAMP client failed to set already installing package statuses", zap.Error(err))
		}
		return errors.New("failed because already installing packages")
	}

	// Retrieve last known status (this should return with minimal info even on first time).
	// If there is a problem retrieving the last saved PackageStatuses, we will log the error
	// but continue on as the only thing missing will be the agent package hash.
	pkgStatuses, err := c.getVerifiedPackageStatuses()
	if err != nil {
		c.logger.Warn("Problem with package statuses on starting install", zap.Error(err))
	}

	lastPkgStatusMap := make(map[string]*protobufs.PackageStatus)
	if pkgStatuses != nil && pkgStatuses.GetPackages() != nil {
		lastPkgStatusMap = pkgStatuses.GetPackages()
	}

	// Loop through all of the available packages sent from the server and create initial PackageStatuses
	for pkgName, availablePkg := range availablePkgs.Packages {
		lastPkgStatus := lastPkgStatusMap[pkgName]
		curPkgStatuses.Packages[pkgName] = c.buildInitialPackageStatus(pkgName, availablePkg, lastPkgStatus)
	}

	// This is an error because we need this file for communication during the update
	if err = c.packagesStateProvider.SetLastReportedStatuses(curPkgStatuses); err != nil {
		return fmt.Errorf("failed to save last reported package statuses: %w", err)
	}

	if err = c.opampClient.SetPackageStatuses(curPkgStatuses); err != nil {
		return fmt.Errorf("opamp client failed to set package statuses: %w", err)
	}

	// Start update if applicable
	if curPkgStatuses.Packages[packagestate.CollectorPackageName].Status == protobufs.PackageStatusEnum_PackageStatusEnum_Installing {
		collectorDownloadableFile := availablePkgs.GetPackages()[packagestate.CollectorPackageName].GetFile()
		c.startCollectorPackageInstall(curPkgStatuses, collectorDownloadableFile)
	}

	return nil
}

// buildInitialCollectorPackageStatus sets up the initial package status message any package
func (c *Client) buildInitialPackageStatus(pkgName string, availablePkg *protobufs.PackageAvailable,
	lastPkgStatus *protobufs.PackageStatus) *protobufs.PackageStatus {
	var initPkgStatus *protobufs.PackageStatus

	switch pkgName {
	case packagestate.CollectorPackageName:
		initPkgStatus = c.buildInitialCollectorPackageStatus(pkgName, availablePkg, lastPkgStatus)
	// If it's not an expected package, return a failed status
	default:
		c.logger.Error(
			"Package update failed because it is not supported",
			zap.String("package", pkgName))

		initPkgStatus = &protobufs.PackageStatus{
			Name:                 pkgName,
			ServerOfferedVersion: availablePkg.GetVersion(),
			ServerOfferedHash:    availablePkg.GetHash(),
			Status:               protobufs.PackageStatusEnum_PackageStatusEnum_InstallFailed,
			ErrorMessage:         "Package not supported",
		}
	}

	return initPkgStatus
}

// buildInitialCollectorPackageStatus sets up the initial package status message for the collector package
func (c *Client) buildInitialCollectorPackageStatus(pkgName string, availablePkg *protobufs.PackageAvailable,
	lastPkgStatus *protobufs.PackageStatus) *protobufs.PackageStatus {
	initPkgStatus := &protobufs.PackageStatus{
		Name:                 pkgName,
		AgentHasVersion:      version.Version(),
		ServerOfferedVersion: availablePkg.GetVersion(),
		ServerOfferedHash:    availablePkg.GetHash(),
		Status:               protobufs.PackageStatusEnum_PackageStatusEnum_Installed,
	}

	// If the new version is the same as the current version we are already installed
	if version.Version() == availablePkg.GetVersion() {
		c.logger.Info("Package update ignored because no new version offered",
			zap.String("package", pkgName))
		initPkgStatus.AgentHasHash = availablePkg.GetHash()

		return initPkgStatus
	}

	// Only grab agentHash from last status if that version matches the current one
	if lastPkgStatus != nil {
		if lastPkgStatus.GetAgentHasVersion() == version.Version() {
			initPkgStatus.AgentHasHash = lastPkgStatus.GetAgentHasHash()
		} else {
			c.logger.Debug(
				fmt.Sprintf(
					"Current version: %s and last reported package status version: %s differ",
					version.Version(),
					lastPkgStatus.GetAgentHasVersion()),
				zap.String("package", pkgName))
		}
	}

	// Bad install if no version is given
	if availablePkg.GetVersion() == "" {
		c.logger.Info("Packaged update failed because no new version detected",
			zap.String("package", pkgName))
		initPkgStatus.ErrorMessage = "Packaged update failed because no new version detected"
		initPkgStatus.Status = protobufs.PackageStatusEnum_PackageStatusEnum_InstallFailed

		return initPkgStatus
	}

	// Bad install if no file is given
	if availablePkg.File == nil {
		c.logger.Info("Packaged update failed because no downloadable file detected",
			zap.String("package", pkgName))
		initPkgStatus.ErrorMessage = "Packaged update failed because no downloadable file detected"
		initPkgStatus.Status = protobufs.PackageStatusEnum_PackageStatusEnum_InstallFailed

		return initPkgStatus
	}

	initPkgStatus.Status = protobufs.PackageStatusEnum_PackageStatusEnum_Installing

	return initPkgStatus
}

// startCollectorPackageInstall attempts to start updating the collector using a new tarball
func (c *Client) startCollectorPackageInstall(curPkgStatuses *protobufs.PackageStatuses, collectorFile *protobufs.DownloadableFile) {
	c.logger.Info("Package update started",
		zap.String("AllPackagesHash", hex.EncodeToString(curPkgStatuses.ServerProvidedAllPackagesHash)),
		zap.String("package", packagestate.CollectorPackageName))
	// Start installing from file if applicable
	if collectorFile != nil {
		c.safeSetUpdatingPackage(true)
		go c.installPackageFromFile(collectorFile)
	} else {
		c.tryToFailPackageInstall("No valid downloadable file found", true)
	}
}

// installPackageFromFile tries to download and extract the given tarball and then start up the new
// Updater binary that was inside of it
func (c *Client) installPackageFromFile(file *protobufs.DownloadableFile) {
	// There should be no reason for us to exit this function unless there is a problem with the Updater's installation
	defer c.safeSetUpdatingPackage(false)

	if fileManagerErr := c.downloadableFileManager.FetchAndExtractArchive(file); fileManagerErr != nil {
		// Remove the update artifacts that may exist, depending on where FetchAndExtractArchive failed.
		c.downloadableFileManager.CleanupArtifacts()
		errMsg := fmt.Sprintf("Failed to download and verify the supplied downloadable file: %s", fileManagerErr)
		c.tryToFailPackageInstall(errMsg, true)

		return
	}

	if monitorErr := c.updaterManager.StartAndMonitorUpdater(); monitorErr != nil {
		// Remove the update artifacts
		c.downloadableFileManager.CleanupArtifacts()
		c.tryToFailPackageInstall(fmt.Sprintf("Failed to run the latest Updater: %s", monitorErr), true)
	}
}

// tryToFailPackageInstall sets PackageStatuses status to failed and error message if we are in the middle of an install.
// The new status will only be immediately sent if explicitly told to
func (c *Client) tryToFailPackageInstall(errMsg string, sendStatusNow bool) {
	// See if we can retrieve the PackageStatuses where the main package is in an installing state
	pkgStatuses, err := c.getVerifiedPackageStatuses()
	if err != nil {
		c.logger.Error("Problem with PackageStatuses", zap.Error(err))
		return
	}

	collectorPackageStatus := pkgStatuses.Packages[packagestate.CollectorPackageName]
	// If we were not installing before the connection, nothing else to do
	if collectorPackageStatus.Status != protobufs.PackageStatusEnum_PackageStatusEnum_Installing {
		return
	}

	// Fail the package install
	c.failPackageInstall(pkgStatuses, errMsg, sendStatusNow)
}

// failPackageInstall sets PackageStatuses status to failed and error message. The new status will only be
// immediately sent if explicitly told to
func (c *Client) failPackageInstall(pkgStatuses *protobufs.PackageStatuses, errMsg string, sendStatusNow bool) {
	// See if we can retrieve the PackageStatuses where the main package is in an installing state
	if pkgStatuses == nil {
		c.logger.Error("Failed to attempt PackageStatuses failure as none were provided")
		return
	}

	collectorPkgStatus, ok := pkgStatuses.Packages[packagestate.CollectorPackageName]
	if !ok {
		c.logger.Error("Failed to attempt PackageStatuses failure as no collector status provided")
		return
	}

	collectorPkgStatus.Status = protobufs.PackageStatusEnum_PackageStatusEnum_InstallFailed
	if collectorPkgStatus.ErrorMessage == "" {
		collectorPkgStatus.ErrorMessage = errMsg
	}

	c.logger.Error(fmt.Sprintf("Package update failed: %s", collectorPkgStatus.ErrorMessage),
		zap.String("package", packagestate.CollectorPackageName))

	if err := c.packagesStateProvider.SetLastReportedStatuses(pkgStatuses); err != nil {
		c.logger.Error("Failed to set failed install package statuses", zap.Error(err))
	}

	// Only send status to Server if this is set. Otherwise it will happen after collector is restarted
	if sendStatusNow {
		if err := c.opampClient.SetPackageStatuses(pkgStatuses); err != nil {
			c.logger.Error("OpAMP client failed to set failed install package statuses", zap.Error(err))
		}
	}
}

// finishPackageInstall sets PackageStatuses status to installed and agent properties to the offered
// server properties.
func (c *Client) finishPackageInstall(pkgStatuses *protobufs.PackageStatuses) {
	c.logger.Info("Package update was successful",
		zap.String("AllPackagesHash", hex.EncodeToString(pkgStatuses.ServerProvidedAllPackagesHash)),
		zap.String("package", packagestate.CollectorPackageName))

	if pkgStatuses == nil {
		c.logger.Error("Failed to set PackageStatuses to installed as none were provided")
		return
	}

	collectorPkgStatus, ok := pkgStatuses.Packages[packagestate.CollectorPackageName]
	if !ok {
		c.logger.Error("Failed to set PackageStatuses to installed as no collector status provided")
		return
	}

	collectorPkgStatus.Status = protobufs.PackageStatusEnum_PackageStatusEnum_Installed
	collectorPkgStatus.AgentHasVersion = version.Version()
	collectorPkgStatus.AgentHasHash = collectorPkgStatus.ServerOfferedHash

	if err := c.packagesStateProvider.SetLastReportedStatuses(pkgStatuses); err != nil {
		c.logger.Error("Failed to set last reported package statuses", zap.Error(err))
	}

	if err := c.opampClient.SetPackageStatuses(pkgStatuses); err != nil {
		c.logger.Error("OpAMP client failed to set package statuses", zap.Error(err))
	}
}

// getVerifiedPackageStatuses returns the last available PackagesStatuses info only if
// the collector package status exists
func (c *Client) getVerifiedPackageStatuses() (*protobufs.PackageStatuses, error) {
	lastPackageStatuses, err := c.packagesStateProvider.LastReportedStatuses()
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve last reported package statuses: %w", err)
	}

	// If we have no info on our collector package, nothing else to do
	if lastPackageStatuses == nil || lastPackageStatuses.Packages == nil || lastPackageStatuses.Packages[packagestate.CollectorPackageName] == nil {
		return nil, errors.New("failed to retrieve last reported package status for collector package")
	}

	return lastPackageStatuses, nil
}

// stopCollectorMonitoring stops monitoring the collector
func (c *Client) stopCollectorMonitoring() {
	c.collectorMntrCancel()
	c.collectorMntrWg.Wait()
}

// startCollectorMonitoring starts a separate goroutine to monitor the collectors status
func (c *Client) startCollectorMonitoring(ctx context.Context) {
	c.collectorMntrCtx, c.collectorMntrCancel = context.WithCancel(ctx)
	c.collectorMntrWg.Add(1)
	go c.monitorCollectorStatus()
}

// monitorCollectorStatus monitors the status of the collector after startup
func (c *Client) monitorCollectorStatus() {
	defer c.collectorMntrWg.Done()
	statusChan := c.collector.Status()
	select {
	case status := <-statusChan:
		switch {
		case status.Panicked:
			// Currently we can't recover from this so we should log a message and exit with an error code.
			// No need to cleanup on shutdown as if no state is left over that would prevent a new process from starting.
			c.logger.Fatal("Collector encountered unrecoverable error", zap.Error(status.Err))
		case status.Err != nil:
			c.logger.Error("Collector unexpectedly stopped running", zap.Error(status.Err))
		case !status.Running:
			c.logger.Error("Collector unexpectedly stopped running")
		}
	case <-c.collectorMntrCtx.Done():
		c.logger.Debug("collector monitor context closed")
		return
	}
}

func (c *Client) safeSetUpdatingPackage(value bool) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.updatingPackage = value
}

func (c *Client) safeGetUpdatingPackage() bool {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	return c.updatingPackage
}

func (c *Client) safeSetDisconnecting(value bool) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.disconnecting = value
}

func (c *Client) safeGetDisconnecting() bool {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	return c.disconnecting
}
