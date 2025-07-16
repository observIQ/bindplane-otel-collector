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

// Package main provides entry point for the updater
package main

import (
	"fmt"
	"log"

	"github.com/observiq/bindplane-otel-collector/updater/internal/logging"
	"github.com/observiq/bindplane-otel-collector/updater/internal/path"
	"github.com/observiq/bindplane-otel-collector/updater/internal/updater"
	"github.com/observiq/bindplane-otel-collector/updater/internal/version"
	"github.com/spf13/pflag"
	"go.uber.org/zap"
)

func main() {
	var showVersion = pflag.BoolP("version", "v", false, "Prints the version of the updater and exits, if specified.")
	pflag.Parse()

	if *showVersion {
		fmt.Println("observiq-otel-collector updater version", version.Version())
		fmt.Println("commit:", version.GitHash())
		fmt.Println("built at:", version.Date())
		return
	}

	// We can't create the zap logger yet, because we don't know the install dir, which is needed
	// to create the logger. So we pass a Nop logger here.
	installDir, err := path.InstallDir(zap.NewNop(), path.DefaultConfigOverrides)
	if err != nil {
		// Can't use "fail" here since we don't know the install directory
		log.Fatalf("Failed to determine install directory: %s", err)
	}

	logger, err := logging.NewLogger(installDir)
	if err != nil {
		log.Fatalf("Failed to create logger: %s\n", err)
	}

	updater, err := updater.NewUpdater(logger, installDir)
	if err != nil {
		logger.Fatal("Failed to create updater", zap.Error(err))
	}

	if err := updater.Update(); err != nil {
		logger.Fatal("Failed to update", zap.Error(err))
	}

	logger.Info("Updater finished successfully")
}
