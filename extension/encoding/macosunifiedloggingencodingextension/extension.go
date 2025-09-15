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

package macosunifiedloggingencodingextension // import "github.com/observiq/bindplane-otel-collector/extension/encoding/macosunifiedloggingencodingextension"

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.uber.org/zap"

	"github.com/observiq/bindplane-otel-collector/extension/encoding/macosunifiedloggingencodingextension/internal/sharedcache"
	"github.com/observiq/bindplane-otel-collector/extension/encoding/macosunifiedloggingencodingextension/internal/uuidtext"
	"github.com/open-telemetry/opentelemetry-collector-contrib/extension/encoding"
)

var (
	_ encoding.LogsUnmarshalerExtension = (*MacosUnifiedLoggingExtension)(nil)
)

// MacosUnifiedLoggingExtension implements the LogsUnmarshalerExtension interface
// for decoding macOS Unified Logging binary data.
type MacosUnifiedLoggingExtension struct {
	logger *zap.Logger
	config *Config
	codec  *macosUnifiedLoggingCodec
}

// UnmarshalLogs unmarshals binary data into OpenTelemetry log records.
// Currently reads binary data but does not decode it - just creates placeholder records.
func (e *MacosUnifiedLoggingExtension) UnmarshalLogs(buf []byte) (plog.Logs, error) {
	return e.codec.UnmarshalLogs(buf)
}

// SetTimesyncRawData configures the timesync raw data for accurate timestamp conversion
func (e *MacosUnifiedLoggingExtension) SetTimesyncRawData(timesyncRawData map[string][]byte) {
	if e.codec != nil {
		e.codec.timesyncRawData = timesyncRawData
		// Parse the raw data and store as structured data
		e.codec.parseTimesyncRawData()
	}
}

// SetUUIDTextRawData configures the UUID text raw data for accurate message parsing
func (e *MacosUnifiedLoggingExtension) SetUUIDTextRawData(uuidTextRawData map[string][]byte) {
	if e.codec != nil {
		e.codec.uuidTextRawData = uuidTextRawData
		// Parse the raw data and store as structured data
		e.codec.parseUUIDTextRawData()
	}
}

// SetDSCRawData configures the DSC (Dynamic Shared Cache) raw data for shared string parsing
func (e *MacosUnifiedLoggingExtension) SetDSCRawData(dscRawData map[string][]byte) {
	if e.codec != nil {
		e.codec.dscRawData = dscRawData
		// Parse the raw data and store as structured data
		e.codec.parseDSCRawData()
	}
}

// GetCacheProvider returns the cache provider for accessing parsed DSC and UUID data
func (e *MacosUnifiedLoggingExtension) GetCacheProvider() *uuidtext.CacheProvider {
	if e.codec != nil {
		return e.codec.cacheProvider
	}
	return nil
}

// Start initializes the extension.
func (e *MacosUnifiedLoggingExtension) Start(_ context.Context, _ component.Host) error {
	e.codec = &macosUnifiedLoggingCodec{
		logger:        e.logger,
		debugMode:     e.config.DebugMode,
		cacheProvider: uuidtext.NewCacheProvider(),
	}
	return nil
}

// Shutdown shuts down the extension.
func (*MacosUnifiedLoggingExtension) Shutdown(context.Context) error {
	return nil
}

// macosUnifiedLoggingCodec handles the actual decoding of macOS Unified Logging binary data.
type macosUnifiedLoggingCodec struct {
	logger          *zap.Logger
	debugMode       bool
	timesyncRawData map[string][]byte                          // Raw timesync data from files
	timesyncData    map[string]*TimesyncBoot                   // Parsed timesync data for accurate timestamp conversion
	uuidTextRawData map[string][]byte                          // Raw UUID text data from files
	uuidTextData    map[string]*uuidtext.UUIDText              // Parsed UUID text data for accurate message parsing
	dscRawData      map[string][]byte                          // Raw DSC data from files
	dscData         map[string]*sharedcache.SharedCacheStrings // Parsed DSC data for shared string parsing
	cacheProvider   *uuidtext.CacheProvider                    // Cache provider for accessing parsed data
}

// UnmarshalLogs reads binary data and parses tracev3 entries into individual log records.
func (c *macosUnifiedLoggingCodec) UnmarshalLogs(buf []byte) (plog.Logs, error) {
	// Check if we have any data
	if len(buf) == 0 {
		return plog.NewLogs(), nil
	}

	// Parse the tracev3 binary data into individual entries using timesync data if available
	entries, err := ParseTraceV3DataWithTimesync(buf, c.timesyncData)
	if err != nil {
		// If parsing fails, fall back to creating a single entry with error info
		logs := plog.NewLogs()
		resourceLogs := logs.ResourceLogs().AppendEmpty()
		scopeLogs := resourceLogs.ScopeLogs().AppendEmpty()
		logRecord := scopeLogs.LogRecords().AppendEmpty()

		logRecord.SetTimestamp(pcommon.NewTimestampFromTime(time.Now()))
		logRecord.SetObservedTimestamp(pcommon.NewTimestampFromTime(time.Now()))
		logRecord.SetSeverityNumber(plog.SeverityNumberError)
		logRecord.SetSeverityText("ERROR")

		message := fmt.Sprintf("Failed to parse tracev3 data (%d bytes): %v", len(buf), err)
		if c.debugMode && len(buf) >= 16 {
			message += fmt.Sprintf(" (first 16 bytes: %x)", buf[:16])
		}

		logRecord.Body().SetStr(message)
		logRecord.Attributes().PutStr("source", "macos_unified_logging")
		logRecord.Attributes().PutInt("data_size", int64(len(buf)))
		logRecord.Attributes().PutBool("decoded", false)
		logRecord.Attributes().PutStr("error", err.Error())

		return logs, nil
	}

	// Convert parsed entries to OpenTelemetry logs
	logs := ConvertTraceV3EntriesToLogs(entries)

	// Add debug information if requested
	if c.debugMode && len(buf) >= 16 {
		// Add a debug entry showing raw binary data
		resourceLogs := logs.ResourceLogs().AppendEmpty()
		scopeLogs := resourceLogs.ScopeLogs().AppendEmpty()
		logRecord := scopeLogs.LogRecords().AppendEmpty()

		logRecord.SetTimestamp(pcommon.NewTimestampFromTime(time.Now()))
		logRecord.SetObservedTimestamp(pcommon.NewTimestampFromTime(time.Now()))
		logRecord.SetSeverityNumber(plog.SeverityNumberDebug)
		logRecord.SetSeverityText("DEBUG")

		message := fmt.Sprintf("Raw tracev3 data: %d bytes, %d entries parsed (first 16 bytes: %x)",
			len(buf), len(entries), buf[:16])
		logRecord.Body().SetStr(message)
		logRecord.Attributes().PutStr("source", "macos_unified_logging_debug")
		logRecord.Attributes().PutInt("data_size", int64(len(buf)))
		logRecord.Attributes().PutInt("parsed_entries", int64(len(entries)))
		logRecord.Attributes().PutBool("decoded", true)
	}

	return logs, nil
}

// parseDSCRawData parses raw DSC data into structured format
func (c *macosUnifiedLoggingCodec) parseDSCRawData() {
	if c.dscRawData == nil {
		return
	}

	if c.dscData == nil {
		c.dscData = make(map[string]*sharedcache.SharedCacheStrings)
	}

	for filePath, rawData := range c.dscRawData {
		// Extract UUID from file path
		uuid := extractDSCUUIDFromPath(filePath)

		// Parse the raw data
		parsedDSC, err := sharedcache.ParseDSC(rawData, uuid)
		if err != nil {
			c.logger.Error("Failed to parse DSC data",
				zap.String("file", filePath),
				zap.String("uuid", uuid),
				zap.Error(err))
			continue
		}

		// Set the DSC UUID from the file path
		parsedDSC.DSCUUID = uuid

		// Store parsed data
		c.dscData[uuid] = parsedDSC

		// Update cache provider with parsed data
		if c.cacheProvider != nil {
			c.cacheProvider.UpdateDSC(uuid, "", parsedDSC)
		}

		c.logger.Debug("Parsed DSC data",
			zap.String("file", filePath),
			zap.String("uuid", uuid),
			zap.Int("numberRanges", int(parsedDSC.NumberRanges)),
			zap.Int("numberUUIDs", int(parsedDSC.NumberUUIDs)))
	}

	c.logger.Info("Completed DSC data parsing",
		zap.Int("totalFiles", len(c.dscRawData)),
		zap.Int("successfullyParsed", len(c.dscData)))
}

// extractDSCUUIDFromPath extracts the UUID from a DSC file path
func extractDSCUUIDFromPath(filePath string) string {
	// Get the base filename (last component of the path)
	filename := filepath.Base(filePath)

	// Remove any extension if present
	filename = strings.TrimSuffix(filename, filepath.Ext(filename))

	// The filename should be the UUID (32 hex characters)
	if len(filename) == 32 {
		// Check if it's all hex (A-F, 0-9)
		for _, char := range filename {
			if !((char >= '0' && char <= '9') || (char >= 'A' && char <= 'F') || (char >= 'a' && char <= 'f')) {
				return filename
			}
		}
		return strings.ToUpper(filename) // Normalize to uppercase
	}

	// If not a standard UUID format, return as-is
	return filename
}

// parseTimesyncRawData parses raw timesync data into structured format
func (c *macosUnifiedLoggingCodec) parseTimesyncRawData() {
	if c.timesyncRawData == nil {
		return
	}

	if c.timesyncData == nil {
		c.timesyncData = make(map[string]*TimesyncBoot)
	}

	for filePath, rawData := range c.timesyncRawData {
		// Parse the raw data
		fileTimesyncData, err := ParseTimesyncData(rawData)
		if err != nil {
			c.logger.Error("Failed to parse timesync data",
				zap.String("file", filePath),
				zap.Error(err))
			continue
		}

		// Merge the timesync data (same logic as before)
		for bootUUID, bootData := range fileTimesyncData {
			if existing, exists := c.timesyncData[bootUUID]; exists {
				// Merge records if boot UUID already exists
				existing.TimesyncRecords = append(existing.TimesyncRecords, bootData.TimesyncRecords...)
			} else {
				c.timesyncData[bootUUID] = bootData
			}
		}

		c.logger.Debug("Parsed timesync data",
			zap.String("file", filePath),
			zap.Int("bootRecords", len(fileTimesyncData)))
	}

	c.logger.Info("Completed timesync data parsing",
		zap.Int("totalFiles", len(c.timesyncRawData)),
		zap.Int("bootUUIDs", len(c.timesyncData)))
}

// parseUUIDTextRawData parses raw UUID text data into structured format
func (c *macosUnifiedLoggingCodec) parseUUIDTextRawData() {
	if c.uuidTextRawData == nil {
		return
	}

	if c.uuidTextData == nil {
		c.uuidTextData = make(map[string]*uuidtext.UUIDText)
	}

	for filePath, rawData := range c.uuidTextRawData {
		// Extract UUID from file path
		uuid := extractUUIDFromPath(filePath)

		// Parse the raw data
		parsedUUIDText, err := uuidtext.ParseUUIDText(rawData, uuid)
		if err != nil {
			c.logger.Error("Failed to parse UUID text data",
				zap.String("file", filePath),
				zap.String("uuid", uuid),
				zap.Error(err))
			continue
		}

		// Set the UUID from the file path
		parsedUUIDText.UUID = uuid

		// Store parsed data
		c.uuidTextData[uuid] = parsedUUIDText

		// Update cache provider with parsed data
		if c.cacheProvider != nil {
			c.cacheProvider.UpdateUUID(uuid, "", parsedUUIDText)
		}

		c.logger.Debug("Parsed UUID text data",
			zap.String("file", filePath),
			zap.String("uuid", uuid),
			zap.Int("numberEntries", int(parsedUUIDText.NumberEntries)),
			zap.Int("footerDataSize", len(parsedUUIDText.FooterData)))
	}

	c.logger.Info("Completed UUID text data parsing",
		zap.Int("totalFiles", len(c.uuidTextRawData)),
		zap.Int("successfullyParsed", len(c.uuidTextData)))
}

// extractUUIDFromPath extracts the UUID from a UUID text file path
func extractUUIDFromPath(filePath string) string {
	// Get the base filename (last component of the path)
	filename := filepath.Base(filePath)

	// Remove any extension if present
	filename = strings.TrimSuffix(filename, filepath.Ext(filename))

	// The filename should be the UUID (32 hex characters)
	if len(filename) == 32 {
		// Check if it's all hex (A-F, 0-9)
		for _, char := range filename {
			if !((char >= '0' && char <= '9') || (char >= 'A' && char <= 'F') || (char >= 'a' && char <= 'f')) {
				return filename
			}
		}
		return strings.ToUpper(filename) // Normalize to uppercase
	}

	// If not a standard UUID format, return as-is
	return filename
}
