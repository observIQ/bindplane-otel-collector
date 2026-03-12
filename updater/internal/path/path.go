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

// Package path contains helper functions, some OS specific, for getting paths involved in updating.
package path

import (
	"path/filepath"
)

// TempDir gets the path to the "tmp" dir, used for staging updates & backups
func TempDir(installDir string) string {
	return filepath.Join(installDir, "tmp")
}

// LatestDir gets the path to the "latest" dir, where the new artifacts are unpacked.
func LatestDir(installDir string) string {
	return filepath.Join(TempDir(installDir), "latest")
}

// BackupDir gets the path to the "rollback" dir, where current artifacts are backed up.
func BackupDir(installDir string) string {
	return filepath.Join(TempDir(installDir), "rollback")
}

// ServiceFileDir gets the directory of the service file definitions
func ServiceFileDir(installDir string) string {
	return filepath.Join(installDir, "install")
}

// BackupServiceFile returns the full path to the backup service file
func BackupServiceFile(installDir string) string {
	return filepath.Join(BackupDir(installDir), "backup.service")
}

// LogFile returns the full path to the log file for the updater
func LogFile(installDir string) string {
	return filepath.Join(installDir, "log", "updater.log")
}
