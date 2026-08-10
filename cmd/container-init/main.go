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

// Package main is a small container-init binary packaged into the collector
// images. It is intended to run as a kubernetes initContainers command to
// prepare a volume for the collector: it recursively creates the parent
// directories of the given config and logging paths and writes default
// collector and logging configs to them.
package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
)

// Default file contents. The collector config is the same minimal nop
// pipeline the container images ship as their default config.yaml, and the
// logging config matches config/logging.stdout.yaml.
const (
	defaultCollectorConfig = `receivers:
  nop:
processors:
  batch:
exporters:
  nop:
service:
  pipelines:
    metrics:
      receivers: [nop]
      processors: [batch]
      exporters: [nop]
  telemetry:
    metrics:
      level: none
`
	defaultLoggingConfig = `output: stdout
level: info
`
)

func main() {
	configPath := flag.String("config", "", "absolute path to write the default collector config (required)")
	loggingPath := flag.String("logging", "", "absolute path to write the default logging config (required)")
	overwrite := flag.Bool("overwrite", false, "overwrite existing files")
	flag.Parse()

	if *configPath == "" || *loggingPath == "" {
		flag.Usage()
		os.Exit(2)
	}

	if err := run(*configPath, *loggingPath, *overwrite); err != nil {
		log.Fatalf("Failed to initialize container: %v", err)
	}
}

// run creates the parent directories of configPath and loggingPath and
// writes default file contents to them. Both paths must be absolute.
// Existing files are left untouched unless overwrite is true.
func run(configPath, loggingPath string, overwrite bool) error {
	files := []struct{ path, contents string }{
		{configPath, defaultCollectorConfig},
		{loggingPath, defaultLoggingConfig},
	}

	// Validate both paths before touching the filesystem.
	for _, f := range files {
		if !filepath.IsAbs(f.path) {
			return fmt.Errorf("path %s must be absolute", f.path)
		}
	}

	for _, f := range files {
		dir := filepath.Dir(f.path)
		if _, err := os.Stat(dir); errors.Is(err, os.ErrNotExist) {
			if err := os.MkdirAll(dir, 0750); err != nil {
				return fmt.Errorf("create directory %s: %w", dir, err)
			}
			log.Printf("created directory %s", dir)
		} else if err != nil {
			return fmt.Errorf("stat %s: %w", dir, err)
		}
		if !overwrite {
			if _, err := os.Stat(f.path); err == nil {
				log.Printf("skipped %s: file already exists", f.path)
				continue
			} else if !errors.Is(err, os.ErrNotExist) {
				return fmt.Errorf("stat %s: %w", f.path, err)
			}
		}
		if err := os.WriteFile(f.path, []byte(f.contents), 0600); err != nil {
			return fmt.Errorf("write %s: %w", f.path, err)
		}
		log.Printf("wrote %s", f.path)
	}
	return nil
}
