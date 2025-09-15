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

package macosunifiedloggingreceiver // import "github.com/observiq/bindplane-otel-collector/receiver/macosunifiedloggingreceiver"

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/receiver"
	"go.opentelemetry.io/collector/receiver/receiverhelper"
	"go.uber.org/zap"

	"github.com/open-telemetry/opentelemetry-collector-contrib/extension/encoding"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/stanza/fileconsumer"

	"github.com/observiq/bindplane-otel-collector/extension/encoding/macosunifiedloggingencodingextension"
)

// macosUnifiedLogReceiver implements receiver.Logs for macOS Unified Logging traceV3 files
type macosUnifiedLogReceiver struct {
	config   *Config
	set      receiver.Settings
	consumer consumer.Logs
	obsrecv  *receiverhelper.ObsReport

	// File consumer for watching and reading traceV3 files
	fileConsumer *fileconsumer.Manager

	// Encoding extension for decoding traceV3 files
	encodingExt encoding.LogsUnmarshalerExtension

	// Context for cancellation
	cancel context.CancelFunc

	// Map to track processed files
	processedFiles map[string]struct{}
	mu             sync.Mutex
}

// newMacOSUnifiedLogReceiver creates a new macOS Unified Logging receiver
func newMacOSUnifiedLogReceiver(
	config *Config,
	set receiver.Settings,
	nextConsumer consumer.Logs,
) (receiver.Logs, error) {
	obsrecv, err := receiverhelper.NewObsReport(receiverhelper.ObsReportSettings{
		ReceiverID:             set.ID,
		ReceiverCreateSettings: set,
	})
	if err != nil {
		return nil, err
	}

	return &macosUnifiedLogReceiver{
		config:         config,
		set:            set,
		consumer:       nextConsumer,
		obsrecv:        obsrecv,
		processedFiles: make(map[string]struct{}),
	}, nil
}

// Start implements receiver.Logs
func (r *macosUnifiedLogReceiver) Start(ctx context.Context, host component.Host) error {
	r.set.Logger.Info("Starting macOS Unified Log receiver")

	// Load the encoding extension
	encodingExt, err := r.loadEncodingExtension(host)
	if err != nil {
		r.set.Logger.Error("Failed to load encoding extension", zap.Error(err))
		return err
	}
	r.encodingExt = encodingExt
	r.set.Logger.Info("Encoding extension loaded successfully")

	// Load timesync data if paths are configured
	if len(r.config.TimesyncPaths) > 0 {
		err := r.loadTimesyncData()
		if err != nil {
			r.set.Logger.Warn("Failed to load timesync data, timestamps may be inaccurate", zap.Error(err))
		} else {
			r.set.Logger.Info("Successfully loaded timesync data for accurate timestamps")
		}
	} else {
		r.set.Logger.Warn("No timesync paths configured, timestamps may be inaccurate")
	}

	// Load UUID text data if paths are configured
	if len(r.config.UUIDTextPaths) > 0 {
		err := r.loadUUIDTextData()
		if err != nil {
			r.set.Logger.Warn("Failed to load UUID text data, message parsing may be inaccurate", zap.Error(err))
		} else {
			r.set.Logger.Info("Successfully loaded UUID text data for accurate message parsing")
		}
	} else {
		r.set.Logger.Warn("No UUID text paths configured, message parsing may be inaccurate")
	}

	// Load DSC data if paths are configured
	if len(r.config.DSCPaths) > 0 {
		err := r.loadDSCData()
		if err != nil {
			r.set.Logger.Warn("Failed to load DSC data, shared string parsing may be inaccurate", zap.Error(err))
		} else {
			r.set.Logger.Info("Successfully loaded DSC data for accurate shared string parsing")
		}
	} else {
		r.set.Logger.Warn("No DSC paths configured, shared string parsing may be inaccurate")
	}

	// Create the file consumer
	fileConsumer, err := r.createFileConsumer(ctx)
	if err != nil {
		r.set.Logger.Error("Failed to create file consumer", zap.Error(err))
		return err
	}
	r.fileConsumer = fileConsumer
	r.set.Logger.Info("File consumer created successfully")

	// Start the file consumer
	err = r.fileConsumer.Start(nil)
	if err != nil {
		r.set.Logger.Error("Failed to start file consumer", zap.Error(err))
		return err
	}
	r.set.Logger.Info("File consumer started successfully")

	return nil
}

// Shutdown implements receiver.Logs
func (r *macosUnifiedLogReceiver) Shutdown(_ context.Context) error {
	if r.cancel != nil {
		r.cancel()
	}

	if r.fileConsumer != nil {
		if err := r.fileConsumer.Stop(); err != nil {
			r.set.Logger.Error("Error stopping file consumer", zap.Error(err))
			return err
		}
	}

	r.set.Logger.Info("macOS Unified Logging receiver stopped")
	return nil
}

// loadEncodingExtension loads the specified encoding extension from the host
func (r *macosUnifiedLogReceiver) loadEncodingExtension(host component.Host) (encoding.LogsUnmarshalerExtension, error) {
	// Parse encoding name as component ID
	var encodingID component.ID
	if err := encodingID.UnmarshalText([]byte(r.config.Encoding)); err != nil {
		return nil, fmt.Errorf("invalid encoding ID %q: %w", r.config.Encoding, err)
	}

	// Get extension from host
	extensions := host.GetExtensions()
	ext, ok := extensions[encodingID]
	if !ok {
		return nil, fmt.Errorf("encoding extension %q not found", r.config.Encoding)
	}

	// Cast to logs unmarshaler extension
	logsUnmarshaler, ok := ext.(encoding.LogsUnmarshalerExtension)
	if !ok {
		return nil, errors.New("extension is not a logs unmarshaler")
	}

	return logsUnmarshaler, nil
}

// createFileConsumer creates and configures the file consumer for traceV3 files
func (r *macosUnifiedLogReceiver) createFileConsumer(_ context.Context) (*fileconsumer.Manager, error) {
	r.set.Logger.Info("Creating file consumer", zap.Any("config", r.config))

	// Get the file consumer configuration
	fcConfig := r.config.getFileConsumerConfig()

	// Create file consumer with our custom emit callback and no encoding (we handle it via extension)
	fileConsumer, err := fcConfig.Build(
		r.set.TelemetrySettings,
		r.consumeTraceV3Tokens,
	)
	if err != nil {
		r.set.Logger.Error("Failed to create file consumer", zap.Error(err))
		return nil, fmt.Errorf("failed to create file consumer: %w", err)
	}

	r.set.Logger.Info("File consumer created successfully")
	return fileConsumer, nil
}

// consumeTraceV3Tokens processes tokens (file chunks) from the file consumer
func (r *macosUnifiedLogReceiver) consumeTraceV3Tokens(ctx context.Context, tokens [][]byte, attributes map[string]any, _ int64, _ []int64) error {
	obsrecvCtx := r.obsrecv.StartLogsOp(ctx)

	// Debug: Log all available attributes
	r.set.Logger.Debug("Received attributes", zap.Any("attributes", attributes))
	r.set.Logger.Info("Processing traceV3 tokens", zap.Int("tokenCount", len(tokens)), zap.Any("attributes", attributes))

	// Get the file path from attributes
	filePath := "unknown"
	if fp, ok := attributes["log.file.path"]; ok {
		filePath = fmt.Sprintf("%v", fp)
	}

	// Check if we've already processed this file
	r.mu.Lock()
	if _, exists := r.processedFiles[filePath]; exists {
		r.mu.Unlock()
		// File already processed, skip
		r.obsrecv.EndLogsOp(obsrecvCtx, "macos_unified_log", 0, nil)
		return nil
	}
	// Mark this file as processed
	r.processedFiles[filePath] = struct{}{}
	r.mu.Unlock()

	// Calculate total size of all tokens for this file
	totalSize := 0
	for _, token := range tokens {
		totalSize += len(token)
	}

	// Process each token (file chunk) through the encoding extension
	var allLogs plog.Logs
	totalLogRecords := 0

	for i, token := range tokens {
		r.set.Logger.Debug("Processing token", zap.Int("tokenIndex", i), zap.Int("tokenSize", len(token)))

		// Use the encoding extension to decode the binary token
		decodedLogs, err := r.encodingExt.UnmarshalLogs(token)
		if err != nil {
			r.set.Logger.Error("Failed to decode token with encoding extension",
				zap.Error(err), zap.Int("tokenIndex", i), zap.Int("tokenSize", len(token)))
			continue
		}

		// Add file metadata as resource attributes to all decoded logs
		for j := 0; j < decodedLogs.ResourceLogs().Len(); j++ {
			resourceLogs := decodedLogs.ResourceLogs().At(j)
			resource := resourceLogs.Resource()
			resource.Attributes().PutStr("log.file.path", filePath)
			resource.Attributes().PutStr("log.file.format", "macos_unified_log_tracev3")
		}

		// If this is the first token, initialize allLogs
		if i == 0 {
			allLogs = decodedLogs
		} else {
			// Append additional logs to the first set
			for j := 0; j < decodedLogs.ResourceLogs().Len(); j++ {
				decodedResourceLogs := decodedLogs.ResourceLogs().At(j)
				decodedResourceLogs.CopyTo(allLogs.ResourceLogs().AppendEmpty())
			}
		}

		totalLogRecords += decodedLogs.LogRecordCount()
	}

	// If no logs were created, create a fallback entry
	if totalLogRecords == 0 {
		allLogs = plog.NewLogs()
		resourceLogs := allLogs.ResourceLogs().AppendEmpty()
		resource := resourceLogs.Resource()
		resource.Attributes().PutStr("log.file.path", filePath)
		resource.Attributes().PutStr("log.file.format", "macos_unified_log_tracev3")

		scopeLogs := resourceLogs.ScopeLogs().AppendEmpty()
		logRecord := scopeLogs.LogRecords().AppendEmpty()
		r.setLogRecordAttributes(&logRecord, totalSize, len(tokens))
		logRecord.Body().SetStr(fmt.Sprintf("No logs decoded from traceV3 file: %s (size: %d bytes, tokens: %d)", filePath, totalSize, len(tokens)))
		totalLogRecords = 1
	}

	// Send all decoded logs to the consumer
	err := r.consumer.ConsumeLogs(ctx, allLogs)
	if err != nil {
		r.set.Logger.Error("Failed to consume logs", zap.Error(err))
	}

	r.set.Logger.Info("Successfully processed traceV3 file",
		zap.String("filePath", filePath),
		zap.Int("totalSize", totalSize),
		zap.Int("tokens", len(tokens)),
		zap.Int("logRecords", totalLogRecords))

	r.obsrecv.EndLogsOp(obsrecvCtx, "macos_unified_log", totalLogRecords, err)

	return nil
}

func (*macosUnifiedLogReceiver) setLogRecordAttributes(logRecord *plog.LogRecord, totalSize, lenTokens int) {
	logRecord.SetTimestamp(pcommon.NewTimestampFromTime(time.Now()))
	logRecord.SetSeverityNumber(plog.SeverityNumberInfo)
	logRecord.SetSeverityText("INFO")

	// Add file information as attributes
	logRecord.Attributes().PutInt("file.total.size", int64(totalSize))
	logRecord.Attributes().PutInt("file.token.count", int64(lenTokens))
	logRecord.Attributes().PutStr("receiver.name", "macosunifiedlog")
}

// loadTimesyncData loads timesync files and configures the encoding extension with the raw data
func (r *macosUnifiedLogReceiver) loadTimesyncData() error {
	timesyncRawData := make(map[string][]byte)

	for _, timesyncPath := range r.config.TimesyncPaths {
		err := r.loadTimesyncFromPath(timesyncPath, timesyncRawData)
		if err != nil {
			r.set.Logger.Error("Failed to load timesync from path", zap.String("path", timesyncPath), zap.Error(err))
			continue
		}
	}

	if len(timesyncRawData) == 0 {
		return fmt.Errorf("no timesync data loaded from any path")
	}

	// Log all loaded timesync files for debugging
	var loadedFiles []string
	totalBytes := 0
	for filePath, data := range timesyncRawData {
		loadedFiles = append(loadedFiles, filePath)
		totalBytes += len(data)
	}

	r.set.Logger.Info("=== RECEIVER DEBUG: TIMESYNC RAW DATA LOADED ===",
		zap.Int("totalTimesyncFiles", len(timesyncRawData)),
		zap.Strings("loadedFiles", loadedFiles),
		zap.Int("totalBytes", totalBytes))

	// Configure the encoding extension with the raw timesync data
	if macosExt, ok := r.encodingExt.(*macosunifiedloggingencodingextension.MacosUnifiedLoggingExtension); ok {
		macosExt.SetTimesyncRawData(timesyncRawData)
		r.set.Logger.Info("Configured encoding extension with timesync raw data",
			zap.Int("timesyncFiles", len(timesyncRawData)),
			zap.Int("totalBytes", totalBytes))
	} else {
		return fmt.Errorf("encoding extension is not of expected type")
	}

	return nil
}

// loadTimesyncFromPath loads timesync data from a specific path (file or glob pattern)
func (r *macosUnifiedLogReceiver) loadTimesyncFromPath(path string, timesyncRawData map[string][]byte) error {
	// Check if path contains wildcards
	if strings.Contains(path, "*") {
		// Handle glob pattern
		matches, err := filepath.Glob(path)
		if err != nil {
			return fmt.Errorf("invalid glob pattern %s: %w", path, err)
		}

		for _, match := range matches {
			err := r.loadTimesyncFile(match, timesyncRawData)
			if err != nil {
				r.set.Logger.Warn("Failed to load timesync file", zap.String("file", match), zap.Error(err))
				continue
			}
		}
	} else {
		// Handle single file
		return r.loadTimesyncFile(path, timesyncRawData)
	}

	return nil
}

// loadTimesyncFile loads a single timesync file as raw bytes
func (r *macosUnifiedLogReceiver) loadTimesyncFile(filePath string, timesyncRawData map[string][]byte) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read timesync file %s: %w", filePath, err)
	}

	// Store the raw timesync data using the file path as the key
	timesyncRawData[filePath] = data

	r.set.Logger.Debug("Loaded timesync file as raw bytes",
		zap.String("file", filePath),
		zap.Int("size", len(data)))

	return nil
}

// loadUUIDTextData loads UUID text files and configures the encoding extension with the raw data
func (r *macosUnifiedLogReceiver) loadUUIDTextData() error {
	uuidTextRawData := make(map[string][]byte)

	for _, uuidTextPath := range r.config.UUIDTextPaths {
		err := r.loadUUIDTextFromPath(uuidTextPath, uuidTextRawData)
		if err != nil {
			r.set.Logger.Error("Failed to load UUID text from path", zap.String("path", uuidTextPath), zap.Error(err))
			continue
		}
	}

	if len(uuidTextRawData) == 0 {
		return fmt.Errorf("no UUID text data loaded from any path")
	}

	// Log all loaded UUID text files for debugging
	var loadedFiles []string
	totalBytes := 0
	for filePath, data := range uuidTextRawData {
		loadedFiles = append(loadedFiles, filePath)
		totalBytes += len(data)
	}

	r.set.Logger.Info("=== RECEIVER DEBUG: UUID TEXT RAW DATA LOADED ===",
		zap.Int("totalUUIDTextFiles", len(uuidTextRawData)),
		zap.Strings("loadedFiles", loadedFiles),
		zap.Int("totalBytes", totalBytes))

	// Configure the encoding extension with the raw UUID text data
	if macosExt, ok := r.encodingExt.(*macosunifiedloggingencodingextension.MacosUnifiedLoggingExtension); ok {
		macosExt.SetUUIDTextRawData(uuidTextRawData)
		r.set.Logger.Info("Configured encoding extension with UUID text raw data",
			zap.Int("uuidTextFiles", len(uuidTextRawData)),
			zap.Int("totalBytes", totalBytes))
	} else {
		return fmt.Errorf("encoding extension is not of expected type")
	}

	return nil
}

// loadUUIDTextFromPath loads UUID text data from a specific path (file or glob pattern)
func (r *macosUnifiedLogReceiver) loadUUIDTextFromPath(path string, uuidTextRawData map[string][]byte) error {
	// Check if path contains wildcards
	if strings.Contains(path, "*") {
		// Handle glob pattern
		matches, err := filepath.Glob(path)
		if err != nil {
			return fmt.Errorf("invalid glob pattern %s: %w", path, err)
		}

		for _, match := range matches {
			err := r.loadUUIDTextFile(match, uuidTextRawData)
			if err != nil {
				r.set.Logger.Warn("Failed to load uuid text file", zap.String("file", match), zap.Error(err))
				continue
			}
		}
	} else {
		// Handle single file
		return r.loadUUIDTextFile(path, uuidTextRawData)
	}

	return nil
}

// loadUUIDTextFile loads a single UUID text file as raw bytes
func (r *macosUnifiedLogReceiver) loadUUIDTextFile(filePath string, uuidTextRawData map[string][]byte) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read UUID text file %s: %w", filePath, err)
	}

	// Store the raw UUID text data using the file path as the key
	uuidTextRawData[filePath] = data

	r.set.Logger.Debug("Loaded UUID text file as raw bytes",
		zap.String("file", filePath),
		zap.Int("size", len(data)))

	return nil
}

// loadDSCData loads DSC files and configures the encoding extension with the raw data
func (r *macosUnifiedLogReceiver) loadDSCData() error {
	dscRawData := make(map[string][]byte)

	for _, dscPath := range r.config.DSCPaths {
		err := r.loadDSCFromPath(dscPath, dscRawData)
		if err != nil {
			r.set.Logger.Error("Failed to load DSC from path", zap.String("path", dscPath), zap.Error(err))
			continue
		}
	}

	if len(dscRawData) == 0 {
		return fmt.Errorf("no DSC data loaded from any path")
	}

	// Log all loaded DSC files for debugging
	var loadedFiles []string
	totalBytes := 0
	for filePath, data := range dscRawData {
		loadedFiles = append(loadedFiles, filePath)
		totalBytes += len(data)
	}

	r.set.Logger.Info("=== RECEIVER DEBUG: DSC RAW DATA LOADED ===",
		zap.Int("totalDSCFiles", len(dscRawData)),
		zap.Strings("loadedFiles", loadedFiles),
		zap.Int("totalBytes", totalBytes))

	// Configure the encoding extension with the raw DSC data
	if macosExt, ok := r.encodingExt.(*macosunifiedloggingencodingextension.MacosUnifiedLoggingExtension); ok {
		macosExt.SetDSCRawData(dscRawData)
		r.set.Logger.Info("Configured encoding extension with DSC raw data",
			zap.Int("dscFiles", len(dscRawData)),
			zap.Int("totalBytes", totalBytes))
	} else {
		return fmt.Errorf("encoding extension is not of expected type")
	}

	return nil
}

// loadDSCFromPath loads DSC data from a specific path (file or glob pattern)
func (r *macosUnifiedLogReceiver) loadDSCFromPath(path string, dscRawData map[string][]byte) error {
	// Check if path contains wildcards
	if strings.Contains(path, "*") {
		// Handle glob pattern
		matches, err := filepath.Glob(path)
		if err != nil {
			return fmt.Errorf("invalid glob pattern %s: %w", path, err)
		}

		for _, match := range matches {
			err := r.loadDSCFile(match, dscRawData)
			if err != nil {
				r.set.Logger.Warn("Failed to load DSC file", zap.String("file", match), zap.Error(err))
				continue
			}
		}
	} else {
		// Handle single file
		return r.loadDSCFile(path, dscRawData)
	}

	return nil
}

// loadDSCFile loads a single DSC file as raw bytes
func (r *macosUnifiedLogReceiver) loadDSCFile(filePath string, dscRawData map[string][]byte) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read DSC file %s: %w", filePath, err)
	}

	// Store the raw DSC data using the file path as the key
	dscRawData[filePath] = data

	r.set.Logger.Debug("Loaded DSC file as raw bytes",
		zap.String("file", filePath),
		zap.Int("size", len(data)))

	return nil
}
