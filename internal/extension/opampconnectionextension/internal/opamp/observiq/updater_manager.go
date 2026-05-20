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

package observiq

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"go.uber.org/zap"
)

const updaterDir = "latest"
const defaultShutdownWaitTimeout = 30 * time.Second

// updaterManager handles working with the Updater binary
//
//go:generate mockery --name updaterManager --filename mock_updater_manager.go --structname MockUpdaterManager --output ../mocks
type updaterManager interface {
	// StartAndMonitorUpdater starts the Updater binary and monitors it for failure
	StartAndMonitorUpdater() error
}

// copyExecutable copies the executable at the input file path to the cwd.
// Returns the output path of the executable.
func copyExecutable(logger *zap.Logger, inputPath, cwd string) (string, error) {
	inputPathClean := filepath.Clean(inputPath)

	inputFile, err := os.Open(inputPathClean)
	if err != nil {
		return "", fmt.Errorf("failed to open updater binary for reading: %w", err)
	}
	defer func() {
		if err := inputFile.Close(); err != nil {
			logger.Error("Failed to close input file", zap.Error(err))
		}
	}()

	// Output path is just whatever the actual file name is (e.g. updater.exe),
	// on top of the CWD. We take the absolute path, because it is needed to actually ensure you can
	// exec a file not on your PATH.
	outputPath, err := filepath.Abs(filepath.Join(cwd, filepath.Base(inputPath)))
	if err != nil {
		return "", fmt.Errorf("failed to get absolute path for output: %w", err)
	}

	outputPathClean := filepath.Clean(outputPath)

	// Remove the file if it already exists, need this for macOS
	if err := os.RemoveAll(outputPathClean); err != nil {
		return "", fmt.Errorf("failed to remove any existing executable: %w", err)
	}

	//#nosec G302 - 0700 instead of 0600 since the executable bit needs to be flipped
	outputFile, err := os.OpenFile(outputPathClean, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0700)
	if err != nil {
		return "", fmt.Errorf("failed to open output file: %w", err)
	}
	defer func() {
		if err := outputFile.Close(); err != nil {
			logger.Error("Failed to close output file", zap.Error(err))
		}
	}()

	if _, err := io.Copy(outputFile, inputFile); err != nil {
		return "", fmt.Errorf("failed to copy executable to output: %w", err)
	}

	return outputPathClean, nil
}
