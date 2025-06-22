package observiq

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"

	"github.com/open-telemetry/opamp-go/client"
	"github.com/open-telemetry/opamp-go/client/types"
	"github.com/open-telemetry/opamp-go/protobufs"
	"go.uber.org/zap"
)

const (
	// RemoteFileCapability is the capability for requesting and reporting remote files
	RemoteFileCapability = "com.bindplane.remotefile"
	// RemoteFileRequestType is the message type for a remote file request message
	RemoteFileRequestType = "requestRemoteFile"
	// RemoteFileReportType is the message type for a remote file report message
	RemoteFileReportType = "reportRemoteFile"
)

type remoteFilesSender struct {
	opampClient client.OpAMPClient
	logger      *zap.Logger
}

func newRemoteFilesSender(opampClient client.OpAMPClient, logger *zap.Logger) *remoteFilesSender {
	return &remoteFilesSender{
		opampClient: opampClient,
		logger:      logger,
	}
}

// RemoteFileRequestCustomMessage represents a request to an agent for a remote file
type RemoteFileRequestCustomMessage struct {
	// FilePath is the path to the remote file
	FilePath string `yaml:"file_path"`

	// SessionID is the identifier that the response should use to match the request with the response.
	SessionID string `yaml:"session_id"`
}

// remoteFileReportCustomMessage respresents a response from the agent to a request for a remote file
type remoteFileReportCustomMessage struct {
	SessionID   string `json:"sessionId"`
	FilePayload []byte `json:"filePayload"`
}

func (s *remoteFilesSender) reportRemoteFile(ctx context.Context, data []byte) {

	var request RemoteFileRequestCustomMessage
	err := json.Unmarshal(data, &request)
	if err != nil {
		s.logger.Error("Failed to unmarshal remote file request", zap.Error(err))
		return
	}
	s.logger.Info("Reporting remote file", zap.String("path", request.FilePath))

	// open the file at data and read it
	file, err := os.Open(request.FilePath)
	if err != nil {
		s.logger.Error("Failed to open file", zap.String("path", request.FilePath), zap.Error(err))
		return
	}
	defer file.Close()

	// read the file
	content, err := io.ReadAll(file)
	if err != nil {
		s.logger.Error("Failed to read file", zap.Error(err))
		return
	}

	report := remoteFileReportCustomMessage{
		SessionID:   request.SessionID,
		FilePayload: content,
	}
	reportBytes, err := json.Marshal(report)
	if err != nil {
		s.logger.Error("Failed to marshal remote file report", zap.Error(err))
		return
	}

	s.logger.Info("Sending file response", zap.String("path", request.FilePath))

	success := false
	for i := 0; i < maxSendRetries; i++ {
		sendingChannel, err := s.opampClient.SendCustomMessage(&protobufs.CustomMessage{
			Capability: RemoteFileCapability,
			Type:       RemoteFileReportType,
			Data:       reportBytes,
		})
		switch {
		case err == nil: // OK
			success = true

		case errors.Is(err, types.ErrCustomMessagePending):
			if i == maxSendRetries-1 {
				// Bail out early, since we aren't going to try to send again
				s.logger.Warn("File response was blocked by other custom messages, skipping...", zap.Int("retries", maxSendRetries))
				break
			}

			select {
			case <-sendingChannel:
				continue
			case <-ctx.Done():
				s.logger.Warn("Context done, stopping file response")
				return
			}
		default:
			s.logger.Error("Failed to report file response", zap.Error(err))
		}
		break
	}
	if !success {
		s.logger.Warn("Failed to send file response after all retries")
	} else {
		s.logger.Info("File response sent", zap.String("path", request.FilePath))
	}
}
