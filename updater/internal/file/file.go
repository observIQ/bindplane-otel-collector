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

// Package file provides helper methods for file operations
package file

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	"go.uber.org/zap"
)

// CopyFileOverwrite copies the file from pathIn to pathOut.
// The output file is created if it does not exist.
// If the output file does exist, it is removed, then written from the input file, preserving the output file's mode.
func CopyFileOverwrite(logger *zap.Logger, pathIn, pathOut string) error {
	fileMode := fs.FileMode(0600)
	pathOutClean := filepath.Clean(pathOut)

	// Try to save existing file's permissions
	outFileInfo, _ := os.Stat(pathOutClean)
	if outFileInfo != nil {
		fileMode = outFileInfo.Mode()
	}

	pathInClean := filepath.Clean(pathIn)
	// If the input file cannot be opened for some reason, do NOT delete the file
	if _, err := os.Stat(pathInClean); err != nil {
		return fmt.Errorf("failed to stat input file: %w", err)
	}

	// Remove old file to prevent issues with mac
	if err := os.Remove(pathOutClean); err != nil {
		logger.Debug("Failed to remove output file", zap.Error(err))
	}

	return copyFileInternal(logger, pathIn, pathOut, os.O_CREATE|os.O_WRONLY, fileMode)
}

// CopyFileNoOverwrite copies the file from pathIn to pathOut, preserving the input file's mode.
// If the output file already exists, this function returns an error.
func CopyFileNoOverwrite(logger *zap.Logger, pathIn, pathOut string) error {
	pathInClean := filepath.Clean(pathIn)

	// Use the new file's permissions and fail if there's an issue (want to fail for backup)
	inFileInfo, err := os.Stat(pathInClean)
	if err != nil {
		return fmt.Errorf("failed to retrieve fileinfo for input file: %w", err)
	}

	// the os.O_EXCL flag will make OpenFile error if the file already exists
	return copyFileInternal(logger, pathIn, pathOut, os.O_EXCL|os.O_CREATE|os.O_WRONLY, inFileInfo.Mode())
}

// CopyFileRollback copies the file to the file from pathIn to pathOut, preserving the input file's mode if possible
// Used to perform a rollback
func CopyFileRollback(logger *zap.Logger, pathIn, pathOut string) error {
	// Default to 0600 if we can't determine the input file's mode
	fileMode := fs.FileMode(0600)
	pathInClean := filepath.Clean(pathIn)
	// Use the backup file's permissions as a backup and don't fail on error (best chance for rollback)
	inFileInfo, err := os.Stat(pathInClean)
	switch {
	case errors.Is(err, os.ErrNotExist):
		return fmt.Errorf("input file does not exist: %w", err)
	case err != nil:
		// Even though we failed to stat, we'll continue in this case to give the best chance
		// of rolling back successfully.
		logger.Error("failed to retrieve fileinfo for input file", zap.Error(err))
	default:
		fileMode = inFileInfo.Mode()
	}

	pathOutClean := filepath.Clean(pathOut)
	// Remove old file to prevent issues with mac
	if err = os.Remove(pathOutClean); err != nil {
		logger.Debug("Failed to remove output file", zap.Error(err))
	}

	return copyFileInternal(logger, pathIn, pathOut, os.O_CREATE|os.O_WRONLY, fileMode)
}

// copyFileInternal copies the file at pathIn to pathOut, using the provided flags and file mode
func copyFileInternal(logger *zap.Logger, pathIn, pathOut string, outFlags int, outMode fs.FileMode) error {
	pathInClean := filepath.Clean(pathIn)

	// Open the input file for reading.
	inFile, err := os.Open(pathInClean)
	if err != nil {
		return fmt.Errorf("failed to open input file: %w", err)
	}
	defer func() {
		err := inFile.Close()
		if err != nil {
			logger.Error("Failed to close input file", zap.Error(err))
		}
	}()

	pathOutClean := filepath.Clean(pathOut)
	// Open the output file, creating it if it does not exist and truncating it.
	//#nosec G304 -- out file is cleaned; this is a general purpose copy function
	outFile, err := os.OpenFile(pathOutClean, outFlags, outMode)
	if err != nil {
		return fmt.Errorf("failed to open output file: %w", err)
	}
	defer func() {
		err := outFile.Close()
		if err != nil {
			logger.Error("Failed to close output file", zap.Error(err))
		}
	}()

	// Copy the input file to the output file.
	if _, err := io.Copy(outFile, inFile); err != nil {
		return fmt.Errorf("failed to copy file: %w", err)
	}
	return nil
}
