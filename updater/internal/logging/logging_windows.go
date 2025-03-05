// Copyright  Bindplane, Inc.
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

package logging

import (
	"fmt"
	"net/url"
	"os"
	"sync"

	"github.com/observiq/bindplane-otel-collector/updater/internal/path"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var registerSinkOnce = &sync.Once{}

// NewLogger returns a new logger, that logs to the log directory relative to installDir.
// It deletes the previous log file, as well.
// NewLogger must only be called once, at the start of the program.
func NewLogger(installDir string) (*zap.Logger, error) {
	// On windows, absolute paths do not work for zap's default sink, so we must register it.
	// see: https://github.com/uber-go/zap/issues/621
	var err error
	registerSinkOnce.Do(func() {
		err = zap.RegisterSink("winfile", newWinFileSink)
	})
	if err != nil {
		return nil, fmt.Errorf("failed to registed windows file sink: %w", err)
	}

	logFile := path.LogFile(installDir)

	err = os.RemoveAll(logFile)
	if err != nil {
		return nil, fmt.Errorf("failed to remove previous log file: %w", err)
	}

	conf := zap.NewProductionConfig()
	conf.OutputPaths = []string{
		"winfile:///" + logFile,
	}
	conf.Level.SetLevel(zapcore.DebugLevel)
	conf.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	prodLogger, err := conf.Build()
	if err != nil {
		return nil, fmt.Errorf("failed to build logger: %w", err)
	}

	return prodLogger, nil
}

// Windows requires a special sink, so that we may properly parse the file path
// See: https://github.com/uber-go/zap/issues/621
func newWinFileSink(u *url.URL) (zap.Sink, error) {
	// Remove leading slash left by url.Parse()
	return os.OpenFile(u.Path[1:], os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0600)
}
