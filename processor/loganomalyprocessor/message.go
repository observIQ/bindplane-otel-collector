// Copyright observIQ, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package loganomalyprocessor

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"errors"
	"time"

	"github.com/open-telemetry/opamp-go/client/types"
	"github.com/open-telemetry/opamp-go/protobufs"
	"github.com/open-telemetry/opentelemetry-collector-contrib/extension/opampcustommessages"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

const (
	anomalyCapability  = "com.observiq.loganomalies"
	anomalyRequestType = "requestAnomalySnapshot"
	anomalyReportType  = "reportAnomalySnapshot"
)

type anomalyRequest struct {
	SessionID string    `yaml:"sessionId"`
	Since     time.Time `yaml:"since,omitempty"`
}

type anomalyResponse struct {
	SessionID        string          `json:"sessionId"`
	TelemetryPayload json.RawMessage `json:"telemetry_payload"`
}

func (p *Processor) processOpAMPMessages(o opampcustommessages.CustomCapabilityHandler) {
	p.wg.Done()
	for {
		select {
		case msg := <-o.Message():
			switch msg.Type {
			case anomalyRequestType:
				p.logger.Debug("got anomaly snapshot request")
				p.processAnomalyRequest(msg)
			default:
				p.logger.Warn("Received message of unknown type.", zap.String("messageType", msg.Type))
			}
		case <-p.doneChan:
			return
		}
	}
}


func (p *Processor) processAnomalyRequest(cm *protobufs.CustomMessage) {
	var req anomalyRequest
	err := yaml.Unmarshal(cm.Data, &req)
	if err != nil {
		p.logger.Error("Got invalid anomaly snapshot request.", zap.Error(err))
		return
	}

	p.stateLock.Lock()

	anomalies := make([]*AnomalyStat, 0, len(p.anomalyBuffer)) // clean this up prob
	for _, anomaly := range p.anomalyBuffer {
		anomalies = append(anomalies, anomaly)
	}
	p.stateLock.Unlock()

	// Create response
	response := anomalyResponse{
		SessionID: req.SessionID,

	}

	// Marshal and compress response
	responseData, err := json.Marshal(response)
	if err != nil {
		p.logger.Error("Could not marshal anomaly snapshot response.", zap.Error(err))
		return
	}

	compressedResponse, err := compress(responseData)
	if err != nil {
		p.logger.Error("Failed to compress anomaly snapshot payload.", zap.Error(err))
		return
	}

	// Send response with retry logic
	for {
		msgSendChan, err := p.customCapabilityHandler.SendMessage(anomalyReportType, compressedResponse)
		switch {
		case err == nil:
			p.logger.Debug("Anomaly snapshot response sent successfully")
			return
		case errors.Is(err, types.ErrCustomMessagePending):
			p.logger.Debug("Custom message pending, will try sending again after message is clear.")
			<-msgSendChan
		default:
			p.logger.Error("Failed to send anomaly snapshot message.", zap.Error(err))
			return
		}
	}
}

func compress(data []byte) ([]byte, error) {
	var b bytes.Buffer
	w := gzip.NewWriter(&b)
	_, err := w.Write(data)
	if err != nil {
		return nil, err
	}

	if err := w.Close(); err != nil {
		return nil, err
	}

	return b.Bytes(), nil
}
