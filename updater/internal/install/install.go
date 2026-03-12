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

// Package install handles installation of new collector artifacts
package install

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/observiq/bindplane-otel-collector/updater/internal/action"
	"github.com/observiq/bindplane-otel-collector/updater/internal/file"
	"github.com/observiq/bindplane-otel-collector/updater/internal/path"
	"github.com/observiq/bindplane-otel-collector/updater/internal/rollback"
	"github.com/observiq/bindplane-otel-collector/updater/internal/service"
	"go.uber.org/zap"
)

// Installer is an interface that performs an Install of a new collector.
//
//go:generate mockery --name Installer --filename mock_installer.go --structname MockInstaller
type Installer interface {
	// Install installs new artifacts over the old ones.
	Install(rollback.Rollbacker) error
}

// archiveInstaller allows you to install files from latestDir into installDir,
// as well as update the service configuration using the "Install" method.
type archiveInstaller struct {
	latestDir  string
	installDir string
	backupDir  string
	svc        service.Service
	logger     *zap.Logger
}

// NewInstaller returns a new instance of an Installer.
func NewInstaller(logger *zap.Logger, installDir string, service service.Service) Installer {
	return &archiveInstaller{
		latestDir:  path.LatestDir(installDir),
		svc:        service,
		installDir: installDir,
		backupDir:  path.BackupDir(installDir),
		logger:     logger.Named("installer"),
	}
}

// Install installs the unpacked artifacts in latestDir to installDir,
// as well as installing the new service file using the installer's Service interface.
// It then starts the service.
func (i archiveInstaller) Install(rb rollback.Rollbacker) error {
	// install files that go to installDirPath to their correct location,
	// excluding any config files (logging.yaml, config.yaml, manager.yaml)
	if err := installFiles(i.logger, i.latestDir, i.installDir, i.backupDir, rb); err != nil {
		return fmt.Errorf("failed to install new files: %w", err)
	}
	i.logger.Debug("Install artifacts copied")

	// Update old service config to new service config
	if err := i.svc.Update(); err != nil {
		return fmt.Errorf("failed to update service: %w", err)
	}
	rb.AppendAction(action.NewServiceUpdateAction(i.logger, i.installDir))
	i.logger.Debug("Updated service configuration")

	// Start service
	if err := i.svc.Start(); err != nil {
		return fmt.Errorf("failed to start service: %w", err)
	}
	rb.AppendAction(action.NewServiceStartAction(i.svc))
	i.logger.Debug("Service started")

	return nil
}

// installFiles moves the file tree rooted at inputPath to installDir,
// skipping configuration files. Appends CopyFileAction-s to the Rollbacker as it copies file.
func installFiles(logger *zap.Logger, inputPath, installDir, backupDir string, rb rollback.Rollbacker) error {
	err := filepath.WalkDir(inputPath, func(inPath string, d fs.DirEntry, err error) error {
		switch {
		case err != nil:
			// if there was an error walking the directory, we want to bail out.
			return err
		case d.IsDir():
			// Skip directories, we'll create them when we get a file in the directory.
			return nil
		case skipConfigFiles(inPath):
			// Found a config file that we should skip copying.
			return nil
		}

		// We want the path relative to the directory we are walking in order to calculate where the file should be
		// mirrored in the destination directory.
		relPath, err := filepath.Rel(inputPath, inPath)
		if err != nil {
			return err
		}

		// use the relative path to get the outPath (where we should write the file), and
		// to get the out directory (which we will create if it does not exist).
		outPath := filepath.Join(installDir, relPath)
		outDir := filepath.Dir(outPath)

		if err := os.MkdirAll(outDir, 0750); err != nil {
			return fmt.Errorf("failed to create dir: %w", err)
		}

		// We create the action record here, because we want to record whether the file exists or not before
		// we open the file (which will end up creating the file).
		cfa, err := action.NewCopyFileAction(logger, relPath, outPath, backupDir)
		if err != nil {
			return fmt.Errorf("failed to create copy file action: %w", err)
		}

		// Record that we are performing copying the file.
		// We record before we actually do the action here because the file may be partially written,
		// and we will want to roll that back if that is the case.
		rb.AppendAction(cfa)

		if err := file.CopyFileOverwrite(logger.Named("copy-file"), inPath, outPath); err != nil {
			return fmt.Errorf("failed to copy file: %w", err)
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to walk latest dir: %w", err)
	}

	return nil
}

// installFile moves new file to output path.
// Appends CopyFileAction-s to the Rollbacker as it copies file.
func installFile(logger *zap.Logger, inPath, installDirPath, backupDirPath string, rb rollback.Rollbacker) error {
	baseInPath := filepath.Base(inPath)

	// use the relative path to get the outPath (where we should write the file), and
	// to get the out directory (which we will create if it does not exist).
	outPath := filepath.Join(installDirPath, baseInPath)
	outDir := filepath.Dir(outPath)

	if err := os.MkdirAll(outDir, 0750); err != nil {
		return fmt.Errorf("failed to create dir: %w", err)
	}

	// We create the action record here, because we want to record whether the file exists or not before
	// we open the file (which will end up creating the file).
	cfa, err := action.NewCopyFileAction(logger, baseInPath, outPath, backupDirPath)
	if err != nil {
		return fmt.Errorf("failed to create copy file action: %w", err)
	}

	// Record that we are performing copying the file.
	// We record before we actually do the action here because the file may be partially written,
	// and we will want to roll that back if that is the case.
	rb.AppendAction(cfa)

	if err := file.CopyFileOverwrite(logger.Named("copy-file"), inPath, outPath); err != nil {
		return fmt.Errorf("failed to copy file: %w", err)
	}

	return nil
}

// skipConfigFiles returns true if the given path is a special config file.
// These files should not be overwritten.
func skipConfigFiles(path string) bool {
	var configFiles = []string{
		"config.yaml",
		"logging.yaml",
		"manager.yaml",
	}

	fileName := filepath.Base(path)

	for _, f := range configFiles {
		if fileName == f {
			return true
		}
	}

	return false
}
