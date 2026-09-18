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
	"io"
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestSeed(t *testing.T) {
	dir := t.TempDir()
	opts := &options{
		configPath:  filepath.Join(dir, "storage", "config.yaml"),
		loggingPath: filepath.Join(dir, "storage", "nested", "logging.yaml"),
	}

	// Creates nested directories and writes defaults.
	if err := seed(opts); err != nil {
		t.Fatalf("seed: %v", err)
	}
	requireFileContents(t, opts.configPath, defaultCollectorConfig)
	requireFileContents(t, opts.loggingPath, defaultLoggingConfig)

	// Existing files are preserved without overwrite.
	if err := os.WriteFile(opts.configPath, []byte("custom"), 0600); err != nil {
		t.Fatalf("write custom config: %v", err)
	}
	if err := seed(opts); err != nil {
		t.Fatalf("seed: %v", err)
	}
	requireFileContents(t, opts.configPath, "custom")

	// Overwrite replaces existing files.
	opts.overwrite = true
	if err := seed(opts); err != nil {
		t.Fatalf("seed with overwrite: %v", err)
	}
	requireFileContents(t, opts.configPath, defaultCollectorConfig)
}

// TestRunChownsAfterSeeding pins the ordering the manifest depends on: the
// files are written first and the chown that follows is what makes them
// readable by the unprivileged collector.
func TestRunChownsAfterSeeding(t *testing.T) {
	storage := filepath.Join(t.TempDir(), "storage")
	opts := &options{
		configPath:  filepath.Join(storage, "config.yaml"),
		loggingPath: filepath.Join(storage, "logging.yaml"),
		chownPath:   storage,
		uid:         testUID,
		gid:         testGID,
	}
	ops := newFakeFS()

	if err := run(opts, ops); err != nil {
		t.Fatalf("run: %v", err)
	}
	for _, path := range []string{storage, opts.configPath, opts.loggingPath} {
		if ops.owners[path] != [2]uint32{testUID, testGID} {
			t.Fatalf("%s owner: got %v", path, ops.owners[path])
		}
	}
}

func TestRunWithoutChownLeavesOwnershipAlone(t *testing.T) {
	dir := t.TempDir()
	opts := &options{
		configPath:  filepath.Join(dir, "config.yaml"),
		loggingPath: filepath.Join(dir, "logging.yaml"),
	}
	ops := newFakeFS()

	if err := run(opts, ops); err != nil {
		t.Fatalf("run: %v", err)
	}
	if len(ops.chowns) != 0 {
		t.Fatalf("unexpected chowns: %v", ops.chowns)
	}
}

func TestParseArgs(t *testing.T) {
	const config = "-config=/etc/otel/storage/config.yaml"
	const logging = "-logging=/etc/otel/storage/logging.yaml"

	t.Run("seed only", func(t *testing.T) {
		opts, err := parseArgs([]string{config, logging}, io.Discard)
		if err != nil {
			t.Fatalf("parseArgs: %v", err)
		}
		if opts.chownPath != "" || opts.uid != 0 || opts.gid != 0 {
			t.Fatalf("unexpected chown options: %+v", opts)
		}
	})

	t.Run("seed and chown", func(t *testing.T) {
		opts, err := parseArgs([]string{
			config, logging, "-chown=/etc/otel/storage", "-uid=1000000000", "-gid=1000000000",
		}, io.Discard)
		if err != nil {
			t.Fatalf("parseArgs: %v", err)
		}
		if opts.chownPath != "/etc/otel/storage" || opts.uid != testUID || opts.gid != testGID {
			t.Fatalf("unexpected options: %+v", opts)
		}
	})

	for name, args := range map[string][]string{
		"missing config":      {logging},
		"missing logging":     {config},
		"relative config":     {"-config=config.yaml", logging},
		"relative logging":    {config, "-logging=logging.yaml"},
		"chown without uid":   {config, logging, "-chown=/etc/otel/storage", "-gid=1000000000"},
		"chown without gid":   {config, logging, "-chown=/etc/otel/storage", "-uid=1000000000"},
		"uid without chown":   {config, logging, "-uid=1000000000", "-gid=1000000000"},
		"root uid":            {config, logging, "-chown=/etc/otel/storage", "-uid=0", "-gid=1000000000"},
		"reserved uid":        {config, logging, "-chown=/etc/otel/storage", "-uid=4294967295", "-gid=1"},
		"relative chown":      {config, logging, "-chown=storage", "-uid=1", "-gid=1"},
		"unclean chown":       {config, logging, "-chown=/etc/otel/../otel/storage", "-uid=1", "-gid=1"},
		"root chown":          {config, logging, "-chown=/", "-uid=1", "-gid=1"},
		"positional argument": {config, logging, "extra"},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := parseArgs(slices.Clone(args), io.Discard); err == nil {
				t.Fatalf("expected an error for %v", args)
			}
		})
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
