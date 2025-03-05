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

package action

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/observiq/bindplane-otel-collector/updater/internal/file"
	"go.uber.org/zap"
)

// CopyFileAction is an action that records a file being copied from FromPath to ToPath
type CopyFileAction struct {
	// FromPathRel is the path where the file originated, relative to the "latest"
	// directory
	FromPathRel string
	// ToPath is the path where the file was written.
	ToPath string
	// FileCreated is a bool that records whether this action had to create a new file or not
	FileCreated bool
	backupDir   string
	logger      *zap.Logger
}

var _ RollbackableAction = (*CopyFileAction)(nil)
var _ fmt.Stringer = (*CopyFileAction)(nil)

// NewCopyFileAction creates a new CopyFileAction that indicates a file was copied from
// fromPathRel into toPath. backupDir is specified for rollback purposes.
// NOTE: This action MUST be created BEFORE the action actually takes place; This allows
// for previous existence of the file to be recorded.
func NewCopyFileAction(logger *zap.Logger, fromPathRel, toPath, backupDir string) (*CopyFileAction, error) {
	fileExists := true
	_, err := os.Stat(toPath)
	switch {
	case errors.Is(err, os.ErrNotExist):
		fileExists = false
	case err != nil:
		return nil, fmt.Errorf("unexpected error stat-ing file: %w", err)
	}

	return &CopyFileAction{
		FromPathRel: fromPathRel,
		ToPath:      toPath,
		// The file will be created if it doesn't already exist
		FileCreated: !fileExists,
		backupDir:   backupDir,
		logger:      logger.Named("copy-file-action"),
	}, nil
}

// Rollback will undo the file copy, by either deleting the file if the file did not originally exist,
// or it will copy the old file in the rollback dir if it already exists.
func (c CopyFileAction) Rollback() error {
	if c.FileCreated {
		// File did not exist before this action.
		// We just need to delete this file.
		return os.RemoveAll(c.ToPath)
	}

	// join the relative path to the backup directory to get the location of the backup path
	backupFilePath := filepath.Join(c.backupDir, c.FromPathRel)
	if err := file.CopyFileRollback(c.logger.Named("copy-file"), backupFilePath, c.ToPath); err != nil {
		return fmt.Errorf("failed to copy file: %w", err)
	}

	return nil
}

func (c CopyFileAction) String() string {
	return fmt.Sprintf("CopyFileAction{FromPathRel: '%s', ToPath: '%s', FileCreated: '%t'}", c.FromPathRel, c.ToPath, c.FileCreated)
}
