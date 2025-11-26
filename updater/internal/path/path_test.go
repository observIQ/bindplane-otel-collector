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

package path

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTempDir(t *testing.T) {
	require.Equal(t, filepath.Join("install", "tmp"), TempDir("install"))
}

func TestLatestDir(t *testing.T) {
	require.Equal(t, filepath.Join("install", "tmp", "latest"), LatestDir("install"))
}

func TestBackupDir(t *testing.T) {
	require.Equal(t, filepath.Join("install", "tmp", "rollback"), BackupDir("install"))
}

func TestServiceFileDir(t *testing.T) {
	require.Equal(t, filepath.Join("install", "install"), ServiceFileDir("install"))
}

func TestBackupServiceFile(t *testing.T) {
	require.Equal(t, filepath.Join("install", "tmp", "rollback", "backup.service"), BackupServiceFile("install"))
}

func TestLogFile(t *testing.T) {
	require.Equal(t, filepath.Join("install", "log", "updater.log"), LogFile("install"))
}
