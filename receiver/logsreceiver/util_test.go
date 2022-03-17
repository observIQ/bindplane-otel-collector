// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package logsreceiver

import (
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/consumer/consumertest"
)

func newTempDir(t *testing.T) string {
	tempDir, err := ioutil.TempDir("", "")
	require.NoError(t, err)

	t.Logf("Temp Dir: %s", tempDir)

	t.Cleanup(func() {
		os.RemoveAll(tempDir)
	})

	return tempDir
}

type recallLogger struct {
	*log.Logger
	written []string
}

func newRecallLogger(t *testing.T, tempDir string) *recallLogger {
	path := filepath.Join(tempDir, "test.log")
	logFile, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
	require.NoError(t, err)

	return &recallLogger{
		Logger:  log.New(logFile, "", 0),
		written: []string{},
	}
}

func (l *recallLogger) log(s string) {
	l.written = append(l.written, s)
	l.Logger.Println(s)
}

func (l *recallLogger) recall() []string {
	defer func() { l.written = []string{} }()
	return l.written
}

// TODO use stateless Convert() from #3125 to generate exact pdata.Logs
// for now, just validate body
func expectLogs(sink *consumertest.LogsSink, expected []string) func() bool {
	return func() bool {
		if sink.LogRecordCount() != len(expected) {
			return false
		}

		found := make(map[string]bool)
		for _, e := range expected {
			found[e] = false
		}

		for _, logs := range sink.AllLogs() {
			illLogs := logs.ResourceLogs().
				At(0).InstrumentationLibraryLogs().
				At(0).LogRecords()

			for i := 0; i < illLogs.Len(); i++ {
				body := illLogs.At(i).Body().StringVal()
				found[body] = true
			}
		}

		for _, v := range found {
			if !v {
				return false
			}
		}

		return true
	}
}
