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
	"regexp"
	"strings"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.uber.org/zap"

	"github.com/observiq/bindplane-otel-collector/extension/encoding/macosunifiedloggingencodingextension/internal/firehose"
	"github.com/observiq/bindplane-otel-collector/extension/encoding/macosunifiedloggingencodingextension/internal/helpers"
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

	// Initialize the regex pattern for printf format parsing
	if err := e.codec.initMessageRegex(); err != nil {
		return fmt.Errorf("failed to initialize message regex: %w", err)
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
	timesyncRawData map[string][]byte               // Raw timesync data from files
	timesyncData    map[string]*TimesyncBoot        // Parsed timesync data for accurate timestamp conversion
	uuidTextRawData map[string][]byte               // Raw UUID text data from files
	uuidTextData    map[string]*uuidtext.UUIDText   // Parsed UUID text data for accurate message parsing
	dscRawData      map[string][]byte               // Raw DSC data from files
	dscData         map[string]*sharedcache.Strings // Parsed DSC data for shared string parsing
	cacheProvider   *uuidtext.CacheProvider         // Cache provider for accessing parsed data
	messageRegex    *regexp.Regexp                  // Compiled regex for printf format parsing
}

// UnmarshalLogs reads binary data and parses tracev3 entries into individual log records.
func (c *macosUnifiedLoggingCodec) UnmarshalLogs(buf []byte) (plog.Logs, error) {
	// Check if we have any data
	if len(buf) == 0 {
		return plog.NewLogs(), nil
	}

	// Parse the unified log data
	logDataResults, err := ParseUnifiedLog(buf)
	if err != nil {
		return plog.NewLogs(), err
	}

	otelLogs := plog.NewLogs()
	// Convert the log data results to OpenTelemetry log records
	// Log structure should be minimal, preserve the log body and attributes

	// Debug: Check what we found
	if c.debugMode {
		c.logger.Info("Parsing results",
			zap.Int("catalogDataCount", len(logDataResults.CatalogData)),
			zap.Int("headerDataCount", len(logDataResults.HeaderData)),
			zap.Int("oversizeDataCount", len(logDataResults.OversizeData)))
	}

	// Process each catalog data entry
	for i, catalogData := range logDataResults.CatalogData {
		// Create a local copy to avoid memory aliasing issues
		localCatalogData := catalogData
		catalogFirehoseData, err := c.processFirehoseData(logDataResults, &localCatalogData, i)
		if err != nil {
			return otelLogs, err
		}
		catalogSimpleDumpData := c.processSimpleDumpData(logDataResults, &localCatalogData, i)
		catalogStatedumpData, err := c.processStatedumpData(logDataResults, &localCatalogData, i)
		if err != nil {
			return otelLogs, err
		}

		if c.debugMode {
			c.logger.Info("Processing catalog data",
				zap.Int("catalogIndex", i),
				zap.Int("firehoseDataCount", len(catalogData.FirehoseData)),
				zap.Int("oversizeDataCount", len(catalogData.OversizeData)))
		}

		for j, firehosePreamble := range catalogData.FirehoseData {

			if c.debugMode {
				c.logger.Info("Processing firehose data",
					zap.Int("firehoseIndex", j),
					zap.Int("publicDataCount", len(firehosePreamble.PublicData)))
			}

			for k, firehoseEntry := range firehosePreamble.PublicData {
				// Create a single log record for each firehose entry
				logRecord := otelLogs.ResourceLogs().AppendEmpty().ScopeLogs().AppendEmpty().LogRecords().AppendEmpty()
				if len(firehoseEntry.Message.ItemInfo) > 0 {
					logRecord.Body().SetStr(firehoseEntry.Message.ItemInfo[0].MessageStrings)
				} else {
					logRecord.Body().SetStr(fmt.Sprintf("No message strings found for firehose entry: %d", k))
				}
				logRecord.SetTimestamp(pcommon.NewTimestampFromTime(catalogFirehoseData[k].Timestamp))
				// Add formatted timestamp (ISO format) as an attribute
				logRecord.Attributes().PutStr("timestamp", catalogFirehoseData[k].Timestamp.Format("2006-01-02T15:04:05.000000000Z"))
				logRecord.Attributes().PutStr("subsystem", catalogFirehoseData[k].Subsystem)
				logRecord.Attributes().PutStr("category", catalogFirehoseData[k].Category)
				logRecord.Attributes().PutStr("process", catalogFirehoseData[k].Process)
				logRecord.Attributes().PutStr("process_image_uuid", catalogFirehoseData[k].ProcessUUID)
				logRecord.Attributes().PutStr("boot_uuid", catalogFirehoseData[k].BootUUID)
				logRecord.Attributes().PutStr("timezone_name", catalogFirehoseData[k].TimezoneName)
				logRecord.Attributes().PutInt("activity_id", int64(catalogFirehoseData[k].ActivityID))
				logRecord.Attributes().PutStr("event_type", catalogFirehoseData[k].EventType.String())
				logRecord.Attributes().PutStr("message_type", catalogFirehoseData[k].LogType.String())
				logRecord.Attributes().PutInt("pid", int64(catalogFirehoseData[k].PID))
				logRecord.Attributes().PutInt("euid", int64(catalogFirehoseData[k].EUID))
				logRecord.Attributes().PutInt("thread_id", int64(catalogFirehoseData[k].ThreadID))
				if len(firehoseEntry.Message.BacktraceStrings) > 0 {
					logRecord.Attributes().PutStr("backtrace", strings.Join(firehoseEntry.Message.BacktraceStrings, "\n"))
				}
				logRecord.Attributes().PutStr("library", catalogFirehoseData[k].Library)
				logRecord.Attributes().PutStr("library_uuid", catalogFirehoseData[k].LibraryUUID)
				logRecord.Attributes().PutStr("process", catalogFirehoseData[k].Process)
			}
		}

		for simpledumpIndex, simpledump := range catalogData.SimpledumpData {
			logRecord := otelLogs.ResourceLogs().AppendEmpty().ScopeLogs().AppendEmpty().LogRecords().AppendEmpty()
			logRecord.Body().SetStr(simpledump.MessageString)
			logRecord.SetTimestamp(pcommon.NewTimestampFromTime(catalogSimpleDumpData[simpledumpIndex].Timestamp))
			logRecord.Attributes().PutStr("timestamp", catalogSimpleDumpData[simpledumpIndex].Timestamp.Format("2006-01-02T15:04:05.000000000Z"))
			logRecord.Attributes().PutStr("subsystem", catalogSimpleDumpData[simpledumpIndex].Subsystem)
			logRecord.Attributes().PutStr("process", catalogSimpleDumpData[simpledumpIndex].Process)
			logRecord.Attributes().PutStr("process_image_uuid", catalogSimpleDumpData[simpledumpIndex].ProcessUUID)
			logRecord.Attributes().PutStr("boot_uuid", catalogSimpleDumpData[simpledumpIndex].BootUUID)
			logRecord.Attributes().PutStr("timezone_name", catalogSimpleDumpData[simpledumpIndex].TimezoneName)
			logRecord.Attributes().PutInt("activity_id", int64(catalogSimpleDumpData[simpledumpIndex].ActivityID))
			logRecord.Attributes().PutStr("event_type", catalogSimpleDumpData[simpledumpIndex].EventType.String())
			logRecord.Attributes().PutStr("message_type", catalogSimpleDumpData[simpledumpIndex].LogType.String())
			logRecord.Attributes().PutInt("pid", int64(catalogSimpleDumpData[simpledumpIndex].PID))
			logRecord.Attributes().PutInt("euid", int64(catalogSimpleDumpData[simpledumpIndex].EUID))
			logRecord.Attributes().PutInt("thread_id", int64(catalogSimpleDumpData[simpledumpIndex].ThreadID))
		}
		for statedumpIndex, statedump := range catalogData.StatedumpData {
			logRecord := otelLogs.ResourceLogs().AppendEmpty().ScopeLogs().AppendEmpty().LogRecords().AppendEmpty()
			logRecord.Body().SetStr(string(statedump.Data))
			logRecord.SetTimestamp(pcommon.NewTimestampFromTime(catalogStatedumpData[statedumpIndex].Timestamp))
			logRecord.Attributes().PutStr("timestamp", catalogStatedumpData[statedumpIndex].Timestamp.Format("2006-01-02T15:04:05.000000000Z"))
			logRecord.Attributes().PutStr("subsystem", catalogStatedumpData[statedumpIndex].Subsystem)
			logRecord.Attributes().PutStr("process", catalogStatedumpData[statedumpIndex].Process)
			logRecord.Attributes().PutStr("process_image_uuid", catalogStatedumpData[statedumpIndex].ProcessUUID)
			logRecord.Attributes().PutStr("boot_uuid", catalogStatedumpData[statedumpIndex].BootUUID)
			logRecord.Attributes().PutStr("timezone_name", catalogStatedumpData[statedumpIndex].TimezoneName)
			logRecord.Attributes().PutInt("activity_id", int64(catalogStatedumpData[statedumpIndex].ActivityID))
			logRecord.Attributes().PutStr("event_type", catalogStatedumpData[statedumpIndex].EventType.String())
			logRecord.Attributes().PutStr("message_type", catalogStatedumpData[statedumpIndex].LogType.String())
			logRecord.Attributes().PutInt("pid", int64(catalogStatedumpData[statedumpIndex].PID))
			logRecord.Attributes().PutInt("euid", int64(catalogStatedumpData[statedumpIndex].EUID))
			logRecord.Attributes().PutInt("thread_id", int64(catalogStatedumpData[statedumpIndex].ThreadID))
		}
	}

	return otelLogs, nil
}

// parseDSCRawData parses raw DSC data into structured format
func (c *macosUnifiedLoggingCodec) parseDSCRawData() {
	if c.dscRawData == nil {
		return
	}

	if c.dscData == nil {
		c.dscData = make(map[string]*sharedcache.Strings)
	}

	for filePath, rawData := range c.dscRawData {
		// Extract UUID from file path
		uuid := extractDSCUUIDFromPath(filePath)

		// Debug logging to understand UUID extraction
		if c.debugMode {
			c.logger.Info("Extracting DSC UUID from path",
				zap.String("filePath", filePath),
				zap.String("extractedUUID", uuid),
				zap.Int("uuidLength", len(uuid)))
		}

		// Parse the raw data
		parsedDSC, err := sharedcache.ParseDSC(rawData)
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

	// Check if the filename is 30 characters (truncated UUID)
	// In this case, we need to reconstruct the full UUID from the directory name
	if len(filename) == 30 {
		// Get the parent directory name
		dir := filepath.Dir(filePath)
		parentDir := filepath.Base(dir)

		// Check if parent directory is 2 hex characters
		if len(parentDir) == 2 {
			// Check if parent directory is all hex
			isHex := true
			for _, char := range parentDir {
				if !((char >= '0' && char <= '9') || (char >= 'A' && char <= 'F') || (char >= 'a' && char <= 'f')) {
					isHex = false
					break
				}
			}

			if isHex {
				// Reconstruct the full UUID by combining directory + filename
				fullUUID := strings.ToUpper(parentDir) + strings.ToUpper(filename)
				fmt.Printf("DEBUG: Reconstructed UUID: %s (length: %d)\n", fullUUID, len(fullUUID))
				return fullUUID
			}
		}
	}

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

// initMessageRegex initializes the regex pattern for printf format parsing
// This is based on the Rust implementation's regex pattern
func (c *macosUnifiedLoggingCodec) initMessageRegex() error {
	// This is the regex pattern from the Rust implementation
	// It matches printf-style format specifiers including Apple-specific extensions
	pattern := `(%(?:(?:\{[^}]+}?)(?:[-+0#]{0,5})(?:\d+|\*)?(?:\.(?:\d+|\*))?(?:h|hh|l|ll|w|I|z|t|q|I32|I64)?[cmCdiouxXeEfgGaAnpsSZP@}]|(?:[-+0 #]{0,5})(?:\d+|\*)?(?:\.(?:\d+|\*))?(?:h|hh|l||q|t|ll|w|I|z|I32|I64)?[cmCdiouxXeEfgGaAnpsSZP@%]))`

	regex, err := regexp.Compile(pattern)
	if err != nil {
		return fmt.Errorf("failed to compile regex for printf format parsing: %w", err)
	}

	c.messageRegex = regex
	return nil
}

// parseTimesyncRawData parses raw timesync data into structured format
func (c *macosUnifiedLoggingCodec) parseTimesyncRawData() {
	if c.timesyncRawData == nil {
		return
	}

	// Ensure logger is available for debug logging
	if c.logger == nil {
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

		// Debug logging to understand UUID extraction
		if c.debugMode {
			c.logger.Info("Extracting UUID from path",
				zap.String("filePath", filePath),
				zap.String("extractedUUID", uuid),
				zap.Int("uuidLength", len(uuid)))
		}

		// Parse the raw data
		parsedUUIDText, err := uuidtext.ParseUUIDText(rawData)
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

	// Check if the filename is 30 characters (truncated UUID)
	// In this case, we need to reconstruct the full UUID from the directory name
	if len(filename) == 30 {
		// Get the parent directory name
		dir := filepath.Dir(filePath)
		parentDir := filepath.Base(dir)

		// Check if parent directory is 2 hex characters
		if len(parentDir) == 2 {
			// Check if parent directory is all hex
			isHex := true
			for _, char := range parentDir {
				if !((char >= '0' && char <= '9') || (char >= 'A' && char <= 'F') || (char >= 'a' && char <= 'f')) {
					isHex = false
					break
				}
			}

			if isHex {
				// Reconstruct the full UUID by combining directory + filename
				fullUUID := strings.ToUpper(parentDir) + strings.ToUpper(filename)
				return fullUUID
			}
		}
	}

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

// processFirehoseData processes the firehose data for a given catalog data entry
func (c *macosUnifiedLoggingCodec) processFirehoseData(ulData *UnifiedLogData, catalogData *UnifiedLogCatalogData, catalogIndex int) ([]LogEntry, error) {
	processedLogs := []LogEntry{}
	for preambleIndex, firehosePreamble := range catalogData.FirehoseData {
		for firehoseEntryIndex, firehoseEntry := range firehosePreamble.PublicData {

			entryContinuousTime := uint64(firehoseEntry.ContinousTimeDelta) | (uint64(firehoseEntry.ContinousTimeDeltaUpper) << 32)
			continuousTime := firehosePreamble.BaseContinuousTime + entryContinuousTime
			timestamp := GetTimestamp(c.timesyncData, ulData.HeaderData[0].BootUUID, continuousTime, firehosePreamble.BaseContinuousTime)

			logData := LogEntry{
				ThreadID:       firehoseEntry.ThreadID,
				PID:            catalogData.CatalogData.GetPID(firehosePreamble.FirstProcID, firehosePreamble.SecondProcID),
				ActivityID:     0,
				Time:           timestamp,
				Timestamp:      helpers.UnixEpochToISO(timestamp),
				LogType:        GetLogType(firehoseEntry.LogType, firehoseEntry.ActivityType),
				EventType:      GetEventType(firehoseEntry.ActivityType),
				EUID:           catalogData.CatalogData.GetEUID(firehosePreamble.FirstProcID, firehosePreamble.SecondProcID),
				BootUUID:       ulData.HeaderData[0].BootUUID,
				TimezoneName:   strings.Split(ulData.HeaderData[0].TimezonePath, "/")[len(strings.Split(ulData.HeaderData[0].TimezonePath, "/"))-1],
				MessageEntries: firehoseEntry.Message.ItemInfo,
			}

			switch firehoseEntry.ActivityType {
			case 0x4:
				logData.ActivityID = uint64(firehoseEntry.FirehoseNonActivity.UnknownActivityID)
				messageData, err := firehose.GetFirehoseNonActivityStrings(
					firehoseEntry.FirehoseNonActivity,
					c.cacheProvider,
					uint64(firehoseEntry.FormatStringLocation),
					firehosePreamble.FirstProcID,
					firehosePreamble.SecondProcID,
					&catalogData.CatalogData)
				if err != nil {
					return processedLogs, err
				}

				logData.Library = messageData.Library
				logData.LibraryUUID = messageData.LibraryUUID
				logData.Process = messageData.Process
				logData.ProcessUUID = messageData.ProcessUUID
				logData.RawMessage = messageData.FormatString

				// if the non-activity log entry has a data ref value then the message strings are stored in an oversize log entry
				var logMessage string
				if firehoseEntry.FirehoseNonActivity.DataRefValue != 0 {
					oversizeStrings := GetOversizeStrings(
						uint32(firehoseEntry.FirehoseNonActivity.DataRefValue),
						firehosePreamble.FirstProcID,
						firehosePreamble.SecondProcID,
						&ulData.OversizeData)
					logMessage, err = firehose.FormatFirehoseLogMessage(messageData.FormatString, oversizeStrings, c.messageRegex)
					if err != nil {
						return processedLogs, err
					}
				} else {
					logMessage, err = firehose.FormatFirehoseLogMessage(messageData.FormatString, firehoseEntry.Message.ItemInfo, c.messageRegex)
					if err != nil {
						return processedLogs, err
					}
				}

				// TODO (after MVP): if we are tracking missing data then add the log data to the missing data
				if strings.Contains(logMessage, "<Missing message data>") {
					if c.debugMode {
						c.logger.Info("Encountered missing message data",
							zap.Int("catalogIndex", catalogIndex),
							zap.Int("preambleIndex", preambleIndex),
							zap.Int("firehoseEntryIndex", firehoseEntryIndex))
					}
					continue
				}

				if len(firehoseEntry.Message.BacktraceStrings) > 0 {
					logData.Message = fmt.Sprintf("Backtrace:\n%s\n%s", strings.Join(firehoseEntry.Message.BacktraceStrings, "\n"), logMessage)
				} else {
					logData.Message = logMessage
				}

				if firehoseEntry.FirehoseNonActivity.SubsystemValue != 0 {
					subsystem := catalogData.CatalogData.GetSubsystem(firehoseEntry.FirehoseNonActivity.SubsystemValue, firehosePreamble.FirstProcID, firehosePreamble.SecondProcID)
					logData.Subsystem = subsystem.Subsystem
					logData.Category = subsystem.Category
				}
			case 0x7:
				// no message data in loss entries
				logData.EventType = EventTypeLoss
				logData.LogType = LogTypeLoss
			case 0x2:
				logData.ActivityID = uint64(firehoseEntry.FirehoseActivity.ActivityID)
				messageData, err := firehose.GetFirehoseActivityStrings(
					firehoseEntry.FirehoseActivity,
					c.cacheProvider,
					uint64(firehoseEntry.FormatStringLocation),
					firehosePreamble.FirstProcID,
					firehosePreamble.SecondProcID,
					&catalogData.CatalogData)
				if err != nil {
					return processedLogs, err
				}

				logData.Library = messageData.Library
				logData.LibraryUUID = messageData.LibraryUUID
				logData.Process = messageData.Process
				logData.ProcessUUID = messageData.ProcessUUID
				logData.RawMessage = messageData.FormatString

				logMessage, err := firehose.FormatFirehoseLogMessage(messageData.FormatString, firehoseEntry.Message.ItemInfo, c.messageRegex)
				if err != nil {
					return processedLogs, err
				}

				// TODO (after MVP): if we are tracking missing data then add the log data to the missing data
				if strings.Contains(logMessage, "<Missing message data>") {
					if c.debugMode {
						c.logger.Info("Encountered missing message data",
							zap.Int("catalogIndex", catalogIndex),
							zap.Int("preambleIndex", preambleIndex),
							zap.Int("firehoseEntryIndex", firehoseEntryIndex))
					}
					continue
				}

				if len(firehoseEntry.Message.BacktraceStrings) > 0 {
					logData.Message = fmt.Sprintf("Backtrace:\n%s\n%s", strings.Join(firehoseEntry.Message.BacktraceStrings, "\n"), logMessage)
				} else {
					logData.Message = logMessage
				}
			case 0x6:
				logData.ActivityID = uint64(firehoseEntry.FirehoseSignpost.UnknownActivityID)
				messageData, err := firehose.GetFirehoseSignpostStrings(
					firehoseEntry.FirehoseSignpost,
					c.cacheProvider,
					uint64(firehoseEntry.FormatStringLocation),
					firehosePreamble.FirstProcID,
					firehosePreamble.SecondProcID,
					&catalogData.CatalogData)
				if err != nil {
					c.logger.Error("Failed to get message string data for firehose signpost log entry",
						zap.Error(err))
				}

				logData.Library = messageData.Library
				logData.LibraryUUID = messageData.LibraryUUID
				logData.Process = messageData.Process
				logData.ProcessUUID = messageData.ProcessUUID
				logData.RawMessage = messageData.FormatString

				var logMessage string
				if firehoseEntry.FirehoseNonActivity.DataRefValue != 0 {
					oversizeStrings := GetOversizeStrings(
						uint32(firehoseEntry.FirehoseNonActivity.DataRefValue),
						firehosePreamble.FirstProcID,
						firehosePreamble.SecondProcID,
						&ulData.OversizeData)
					logMessage, err = firehose.FormatFirehoseLogMessage(messageData.FormatString, oversizeStrings, c.messageRegex)
					if err != nil {
						return processedLogs, err
					}
				} else {
					logMessage, err = firehose.FormatFirehoseLogMessage(messageData.FormatString, firehoseEntry.Message.ItemInfo, c.messageRegex)
					if err != nil {
						return processedLogs, err
					}
				}

				// TODO (after MVP): if we are tracking missing data then add the log data to the missing data
				if strings.Contains(logMessage, "<Missing message data>") {
					if c.debugMode {
						c.logger.Info("Encountered missing message data",
							zap.Int("catalogIndex", catalogIndex),
							zap.Int("preambleIndex", preambleIndex),
							zap.Int("firehoseEntryIndex", firehoseEntryIndex))
					}
					continue
				}

				logMessage = fmt.Sprintf("Signpost ID: %X - Signpost Name: %X\n%s", firehoseEntry.FirehoseSignpost.SignpostID, firehoseEntry.FirehoseSignpost.SignpostName, logMessage)

				if len(firehoseEntry.Message.BacktraceStrings) > 0 {
					logData.Message = fmt.Sprintf("Backtrace:\n%s\n%s", strings.Join(firehoseEntry.Message.BacktraceStrings, "\n"), logMessage)
				} else {
					logData.Message = logMessage
				}

				if firehoseEntry.FirehoseSignpost.Subsystem != 0 {
					subsystem := catalogData.CatalogData.GetSubsystem(firehoseEntry.FirehoseSignpost.Subsystem, firehosePreamble.FirstProcID, firehosePreamble.SecondProcID)
					logData.Subsystem = subsystem.Subsystem
					logData.Category = subsystem.Category
				}

			case 0x3:
				messageData, err := firehose.GetFirehoseTraceStrings(
					c.cacheProvider,
					uint64(firehoseEntry.FormatStringLocation),
					firehosePreamble.FirstProcID,
					firehosePreamble.SecondProcID,
					&catalogData.CatalogData)
				if err != nil {
					return processedLogs, err
				}

				logData.Library = messageData.Library
				logData.LibraryUUID = messageData.LibraryUUID
				logData.Process = messageData.Process
				logData.ProcessUUID = messageData.ProcessUUID

				var logMessage string
				logMessage, err = firehose.FormatFirehoseLogMessage(messageData.FormatString, firehoseEntry.Message.ItemInfo, c.messageRegex)
				if err != nil {
					return processedLogs, err
				}

				// TODO (after MVP): if we are tracking missing data then add the log data to the missing data
				if strings.Contains(logMessage, "<Missing message data>") {
					if c.debugMode {
						c.logger.Info("Encountered missing message data",
							zap.Int("catalogIndex", catalogIndex),
							zap.Int("preambleIndex", preambleIndex),
							zap.Int("firehoseEntryIndex", firehoseEntryIndex))
					}
					continue
				}

				if len(firehoseEntry.Message.BacktraceStrings) > 0 {
					logData.Message = fmt.Sprintf("Backtrace:\n%s\n%s", strings.Join(firehoseEntry.Message.BacktraceStrings, "\n"), logMessage)
				} else {
					logData.Message = logMessage
				}

				if firehoseEntry.FirehoseTrace.UnknownPCID != 0 {
				}
			default:
				return processedLogs, fmt.Errorf("parsed unknown firehose log data: %v", firehoseEntry)
			}
			processedLogs = append(processedLogs, logData)
		}
	}
	return processedLogs, nil
}

func (c *macosUnifiedLoggingCodec) processSimpleDumpData(ulData *UnifiedLogData, catalogData *UnifiedLogCatalogData, catalogIndex int) []LogEntry {
	processedLogs := []LogEntry{}
	for _, simpledump := range catalogData.SimpledumpData {
		timestamp := GetTimestamp(c.timesyncData, ulData.HeaderData[0].BootUUID, simpledump.ContinuousTime, uint64(1))
		logEntry := LogEntry{
			Subsystem:    simpledump.Subsystem,
			ThreadID:     simpledump.ThreadID,
			PID:          simpledump.FirstProcID,
			Time:         timestamp,
			Timestamp:    helpers.UnixEpochToISO(timestamp),
			LogType:      LogTypeSimpledump,
			Message:      simpledump.MessageString,
			EventType:    EventTypeSimpledump,
			BootUUID:     ulData.HeaderData[0].BootUUID,
			TimezoneName: strings.Split(ulData.HeaderData[0].TimezonePath, "/")[len(strings.Split(ulData.HeaderData[0].TimezonePath, "/"))-1],
			LibraryUUID:  simpledump.SenderUUID,
			ProcessUUID:  simpledump.DSCSharedCacheUUID,
		}
		processedLogs = append(processedLogs, logEntry)
	}
	return processedLogs
}

func (c *macosUnifiedLoggingCodec) processStatedumpData(ulData *UnifiedLogData, catalogData *UnifiedLogCatalogData, catalogIndex int) ([]LogEntry, error) {
	processedLogs := []LogEntry{}
	for _, statedump := range catalogData.StatedumpData {
		var dataString string
		switch statedump.UnknownDataType {
		case 0x1:
			dataString = ParseStateDumpPlist(statedump.Data)
		case 0x2:
			c.logger.Info("Processing statedump protobuf data", zap.Int("catalogIndex", catalogIndex), zap.ByteString("data", statedump.Data))
			// if protobufMap, err := extractProtobuf(statedump.Data); err == nil {
			// 	if jsonBytes, err := json.Marshal(protobufMap); err == nil {
			// 		dataString = string(jsonBytes)
			// 	} else {
			// 		dataString = "Failed to serialize Protobuf HashMap"
			// 	}
			// } else {
			// 	dataString = fmt.Sprintf("Failed to parse StateDump protobuf: %s", encodeStandard(statedump.Data))
			// }
		case 0x3:
			dataString = ParseStateDumpObject(statedump.Data, statedump.TitleName)
		default:
			c.logger.Warn("Unknown statedump data type", zap.Int("type", int(statedump.UnknownDataType)))
			results, err := helpers.ExtractString(statedump.Data)
			if err != nil {
				return processedLogs, err
			}
			dataString = results
		}

		timestamp := GetTimestamp(c.timesyncData, ulData.HeaderData[0].BootUUID, statedump.ContinuousTime, uint64(1))
		logEntry := LogEntry{
			PID:          statedump.FirstProcID,
			ActivityID:   statedump.ActivityID,
			Time:         timestamp,
			Timestamp:    helpers.UnixEpochToISO(timestamp),
			EventType:    EventTypeStatedump,
			Message:      fmt.Sprintf("title: %s\nObject Library: %s\nObject Type: %s\n%s", statedump.TitleName, statedump.DecoderLibrary, statedump.DecoderType, dataString),
			LogType:      LogTypeStatedump,
			BootUUID:     ulData.HeaderData[0].BootUUID,
			TimezoneName: strings.Split(ulData.HeaderData[0].TimezonePath, "/")[len(strings.Split(ulData.HeaderData[0].TimezonePath, "/"))-1],
		}
		processedLogs = append(processedLogs, logEntry)
	}
	return processedLogs, nil
}
