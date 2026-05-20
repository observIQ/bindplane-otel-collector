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
//
// go:build !windows

package packagestate

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/open-telemetry/opamp-go/protobufs"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestLoadStatusesLinux(t *testing.T) {
	testCases := []struct {
		desc     string
		testFunc func(*testing.T)
	}{
		{
			desc: "Problem reading file",
			testFunc: func(t *testing.T) {
				tmpDir := t.TempDir()
				cantReadJSON := filepath.Join(tmpDir, "noread.json")
				os.WriteFile(cantReadJSON, nil, 0000)
				logger := zap.NewNop()
				p := &FileStateManager{
					logger:   logger,
					jsonPath: cantReadJSON,
				}

				actual, err := p.LoadStatuses()

				assert.ErrorContains(t, err, "failed to read package statuses json:")
				assert.Nil(t, actual)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, tc.testFunc)
	}
}

func TestSaveStatusesLinux(t *testing.T) {
	testCases := []struct {
		desc     string
		testFunc func(*testing.T)
	}{
		{
			desc: "Can't write to file",
			testFunc: func(t *testing.T) {
				tmpDir := t.TempDir()
				os.Chmod(tmpDir, 0400)
				testJSON := filepath.Join(tmpDir, "test.json")
				logger := zap.NewNop()
				p := &FileStateManager{
					logger:   logger,
					jsonPath: testJSON,
				}

				packageStatuses := &protobufs.PackageStatuses{
					ServerProvidedAllPackagesHash: []byte("hash"),
				}

				err := p.SaveStatuses(packageStatuses)

				assert.ErrorContains(t, err, "failed to write package statuses json:")

				// Right now the following code won't work, because the file can't be deleted as we don't have write permissions.
				// It would be nice to have a way to test a write failure, while still being able to delete the file.
				// exists := true
				// if _, err = os.Stat(testJSON); os.IsNotExist(err) {
				// 	exists = false
				// }
				// assert.False(t, exists)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, tc.testFunc)
	}
}
