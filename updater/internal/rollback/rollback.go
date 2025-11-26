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

// Package rollback provides structures and interfaces for rolling back actions
package rollback

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/observiq/bindplane-otel-collector/updater/internal/action"
	"github.com/observiq/bindplane-otel-collector/updater/internal/file"
	"github.com/observiq/bindplane-otel-collector/updater/internal/path"
	"github.com/observiq/bindplane-otel-collector/updater/internal/service"
	"go.uber.org/zap"
)

// Rollbacker is an interface that performs rollback/backup actions.
//
//go:generate mockery --name Rollbacker --filename mock_rollbacker.go --structname MockRollbacker
type Rollbacker interface {
	// AppendAction saves the action so that it can be rolled back later.
	AppendAction(action action.RollbackableAction)
	// Backup backs up the current installation
	Backup() error
	// Rollback undoes the actions recorded by AppendAction.
	Rollback()
}

// filesystemRollbacker is a struct that records rollback information,
// and can use that information to perform a rollback using files backed up
// on the filesystem.
type filesystemRollbacker struct {
	originalSvc service.Service
	backupDir   string
	installDir  string
	actions     []action.RollbackableAction
	logger      *zap.Logger
}

// NewRollbacker returns a new Rollbacker
func NewRollbacker(logger *zap.Logger, installDir string) Rollbacker {
	namedLogger := logger.Named("rollbacker")

	return &filesystemRollbacker{
		backupDir:   path.BackupDir(installDir),
		installDir:  installDir,
		logger:      namedLogger,
		originalSvc: service.NewService(namedLogger, installDir),
	}
}

// AppendAction records the action that was performed, so that it may be undone later.
func (r *filesystemRollbacker) AppendAction(action action.RollbackableAction) {
	r.actions = append(r.actions, action)
}

// Backup backs up the installDir to the rollbackDir
func (r filesystemRollbacker) Backup() error {
	r.logger.Debug("Backing up current installation")
	// Remove any pre-existing backup
	if err := os.RemoveAll(r.backupDir); err != nil {
		return fmt.Errorf("failed to remove previous backup: %w", err)
	}

	// Copy all the files in the install directory to the backup directory
	if err := backupFiles(r.logger, r.installDir, r.backupDir); err != nil {
		return fmt.Errorf("failed to copy files to backup dir: %w", err)
	}

	// Backup the service configuration so we can reload it in case of rollback
	if err := r.originalSvc.Backup(); err != nil {
		return fmt.Errorf("failed to backup service configuration: %w", err)
	}

	return nil
}

// Rollback performs a rollback by undoing all recorded actions.
func (r filesystemRollbacker) Rollback() {
	r.logger.Debug("Performing rollback")
	// We need to loop through the actions slice backwards, to roll back the actions in the correct order.
	// e.g. if StartService was called last, we need to stop the service first, then rollback previous actions.
	for i := len(r.actions) - 1; i >= 0; i-- {
		action := r.actions[i]
		r.logger.Debug("Rolling back action", zap.Any("action", action))
		if err := action.Rollback(); err != nil {
			r.logger.Error("Failed to run rollback action", zap.Error(err))
		}
	}
}

// backupFiles copies files from installDir to output path, skipping tmpDir.
func backupFiles(logger *zap.Logger, installDir, outputPath string) error {
	absTmpDir, err := filepath.Abs(path.TempDir(installDir))
	if err != nil {
		return fmt.Errorf("failed to get absolute path for temporary directory: %w", err)
	}

	err = filepath.WalkDir(installDir, func(inPath string, d fs.DirEntry, err error) error {

		fullPath, absErr := filepath.Abs(inPath)
		if absErr != nil {
			return fmt.Errorf("failed to determine absolute path of file: %w", absErr)
		}

		switch {
		case err != nil:
			// if there was an error walking the directory, we want to bail out.
			return err
		case d.IsDir() && strings.HasPrefix(fullPath, absTmpDir):
			// If this is the "tmp" directory, we want to skip copying this directory,
			// since this folder is only for temporary files (and is where this binary is running right now)
			return filepath.SkipDir
		case d.IsDir():
			// Skip directories, we'll create them when we get a file in the directory.
			return nil
		}

		// We want the path relative to the directory we are walking in order to calculate where the file should be
		// mirrored in the output directory.
		relPath, err := filepath.Rel(installDir, inPath)
		if err != nil {
			return err
		}

		// use the relative path to get the outPath (where we should write the file), and
		// to get the out directory (which we will create if it does not exist).
		outPath := filepath.Join(outputPath, relPath)
		outDir := filepath.Dir(outPath)

		if err := os.MkdirAll(outDir, 0750); err != nil {
			return fmt.Errorf("failed to create dir: %w", err)
		}

		// Fail if copying the input file to the output file would fail
		if err := file.CopyFileNoOverwrite(logger.Named("copy-file"), inPath, outPath); err != nil {
			return fmt.Errorf("failed to copy file: %w", err)
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to walk latest dir: %w", err)
	}

	return nil
}

// backupFile copies original file to output path
func backupFile(logger *zap.Logger, inPath, outputDirPath string) error {
	baseInPath := filepath.Base(inPath)

	// use the relative path to get the outPath (where we should write the file), and
	// to get the out directory (which we will create if it does not exist).
	outPath := filepath.Join(outputDirPath, baseInPath)
	outDir := filepath.Dir(outPath)

	if err := os.MkdirAll(outDir, 0750); err != nil {
		return fmt.Errorf("failed to create dir: %w", err)
	}

	// Fail if copying the input file to the output file would fail
	if err := file.CopyFileNoOverwrite(logger.Named("copy-file"), inPath, outPath); err != nil {
		return fmt.Errorf("failed to copy file: %w", err)
	}

	return nil
}
