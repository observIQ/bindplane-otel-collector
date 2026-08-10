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

package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRun(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "storage", "config.yaml")
	loggingPath := filepath.Join(dir, "storage", "nested", "logging.yaml")

	// Creates nested directories and writes defaults.
	if err := run(configPath, loggingPath, false); err != nil {
		t.Fatalf("run: %v", err)
	}
	requireFileContents(t, configPath, defaultCollectorConfig)
	requireFileContents(t, loggingPath, defaultLoggingConfig)

	// Existing files are preserved without overwrite.
	if err := os.WriteFile(configPath, []byte("custom"), 0600); err != nil {
		t.Fatalf("write custom config: %v", err)
	}
	if err := run(configPath, loggingPath, false); err != nil {
		t.Fatalf("run: %v", err)
	}
	requireFileContents(t, configPath, "custom")

	// Overwrite replaces existing files.
	if err := run(configPath, loggingPath, true); err != nil {
		t.Fatalf("run with overwrite: %v", err)
	}
	requireFileContents(t, configPath, defaultCollectorConfig)
}

func TestRunRejectsRelativePaths(t *testing.T) {
	if err := run("relative/config.yaml", "/abs/logging.yaml", false); err == nil {
		t.Fatal("expected error for relative config path")
	}
	if err := run("/abs/config.yaml", "relative/logging.yaml", false); err == nil {
		t.Fatal("expected error for relative logging path")
	}
}

func requireFileContents(t *testing.T, path, expected string) {
	t.Helper()
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	if string(contents) != expected {
		t.Fatalf("unexpected contents of %s: got %q, want %q", path, contents, expected)
	}
}
