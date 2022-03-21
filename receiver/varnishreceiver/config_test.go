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

package varnishreceiver // import "github.com/observiq/observiq-otel-collector/receiver/varnishreceiver"

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidate(t *testing.T) {
	testCases := []struct {
		desc        string
		cfg         Config
		expectedErr error
	}{
		{
			desc:        "empty config",
			cfg:         Config{},
			expectedErr: nil,
		},
		{
			desc: "missing exec dir",
			cfg: Config{
				ExecDir: "missing/exec",
			},
			expectedErr: fmt.Errorf(errExecDirNotExist.Error(), "stat missing/exec: no such file or directory"),
		},
		{
			desc: "missing working dir",
			cfg: Config{
				WorkingDir: "missing/working",
			},
			expectedErr: fmt.Errorf(errWorkingDirNotExist.Error(), "stat missing/working: no such file or directory"),
		},
		{
			desc: "missing exec and working dir",
			cfg: Config{
				WorkingDir: "missing/working",
				ExecDir:    "missing/exec",
			},
			expectedErr: fmt.Errorf("\"working_dir\" does not exists \"stat missing/working: no such file or directory\"; \"exec_dir\" does not exists \"stat missing/exec: no such file or directory\""),
		},
		{
			desc: "valid exec and working dir",
			cfg: Config{
				WorkingDir: "config_test.go",
				ExecDir:    "config_test.go",
			},
			expectedErr: nil,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			actualErr := tc.cfg.Validate()
			if tc.expectedErr != nil {
				require.EqualError(t, actualErr, tc.expectedErr.Error())
			} else {
				require.NoError(t, actualErr)
			}
		})
	}
}
