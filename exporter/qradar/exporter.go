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

package qradar

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"

	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.uber.org/zap"
)

type qradarExporter struct {
	cfg       *Config
	logger    *zap.Logger
	marshaler logMarshaler
	endpoint  string
	qradarClient
}

// qradarClient is a client for creating connections to IBM QRadar. (created for overriding in tests)
//
//go:generate mockery --name qradarClient --output ./internal/mocks --with-expecter --filename qradar_client.go --structname MockForwarderClient
type qradarClient interface {
	Dial(network string, address string) (net.Conn, error)
	DialWithTLS(network string, addr string, config *tls.Config) (*tls.Conn, error)
	OpenFile(name string) (*os.File, error)
}

type defaultClient struct {
}

func (fc *defaultClient) Dial(network string, address string) (net.Conn, error) {
	return net.Dial(network, address)
}

func (fc *defaultClient) DialWithTLS(network string, addr string, config *tls.Config) (*tls.Conn, error) {
	return tls.Dial(network, addr, config)
}

func (fc *defaultClient) OpenFile(name string) (*os.File, error) {
	cleanPath := filepath.Clean(name)
	return os.OpenFile(cleanPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
}

func newExporter(cfg *Config, params exporter.Settings) (*qradarExporter, error) {
	return &qradarExporter{
		cfg:          cfg,
		logger:       params.Logger,
		marshaler:    newMarshaler(*cfg, params.TelemetrySettings),
		qradarClient: &defaultClient{},
	}, nil
}

func (ce *qradarExporter) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{MutatesData: false}
}

func (ce *qradarExporter) logsDataPusher(ctx context.Context, ld plog.Logs) error {
	// Open connection or file before sending each payload
	writer, err := ce.openWriter(ctx)
	if err != nil {
		return fmt.Errorf("open writer: %w", err)
	}
	defer writer.Close()

	payloads, err := ce.marshaler.MarshalRawLogs(ctx, ld)
	if err != nil {
		return fmt.Errorf("marshal logs: %w", err)
	}

	for _, payload := range payloads {
		if err := ce.send(payload, writer); err != nil {
			return fmt.Errorf("upload to QRadar: %w", err)
		}
	}

	return nil
}

func (ce *qradarExporter) openWriter(ctx context.Context) (io.WriteCloser, error) {
	return ce.openSyslogWriter(ctx)
}

func (ce *qradarExporter) openSyslogWriter(ctx context.Context) (io.WriteCloser, error) {
	var conn net.Conn
	var err error
	transportStr := string(ce.cfg.Syslog.AddrConfig.Transport)

	if ce.cfg.Syslog.TLSSetting != nil {
		var tlsConfig *tls.Config
		tlsConfig, err = ce.cfg.Syslog.TLSSetting.LoadTLSConfig(ctx)
		if err != nil {
			return nil, fmt.Errorf("load TLS config: %w", err)
		}
		conn, err = ce.DialWithTLS(transportStr, ce.cfg.Syslog.AddrConfig.Endpoint, tlsConfig)

		if err != nil {
			return nil, fmt.Errorf("dial with tls: %w", err)
		}
	} else {
		conn, err = ce.Dial(transportStr, ce.cfg.Syslog.AddrConfig.Endpoint)

		if err != nil {
			return nil, fmt.Errorf("dial: %w", err)
		}
	}

	return conn, nil
}

func (ce *qradarExporter) send(msg string, writer io.WriteCloser) error {
	if !strings.HasSuffix(msg, "\n") {
		msg = fmt.Sprintf("%s%s", msg, "\n")
	}

	_, err := io.WriteString(writer, msg)
	return err
}
