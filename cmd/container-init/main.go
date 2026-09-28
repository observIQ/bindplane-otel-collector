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
//
// With -chown it also hands the volume to the unprivileged user the collector
// runs as, which Kubernetes cannot do itself because fsGroup does not apply to
// hostPath volumes. That mode must run as root with CAP_CHOWN and
// CAP_DAC_READ_SEARCH. The chown runs after the files are written, so they end
// up owned by the collector rather than by root.
package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
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
	chownPath := flag.String("chown", "", "absolute path to recursively chown to -uid:-gid after writing the files")
	uid := flag.Uint("uid", 0, "owner uid for -chown")
	gid := flag.Uint("gid", 0, "owner gid for -chown")
	flag.Parse()

	if *configPath == "" || *loggingPath == "" {
		flag.Usage()
		os.Exit(2)
	}
	target, err := newChownTarget(*chownPath, *uid, *gid)
	if err != nil {
		fmt.Fprintf(os.Stderr, "container-init: %v\n", err)
		flag.Usage()
		os.Exit(2)
	}

	if desc := describeProcess(); desc != "" {
		log.Printf("running as %s", desc)
	}

	if err := run(*configPath, *loggingPath, *overwrite); err != nil {
		log.Fatalf("Failed to initialize container: %v", err)
	}
	// After run, so the files it wrote are handed over as well.
	if target.path != "" {
		if err := chownTree(target.path, target.uid, target.gid, os.Lchown); err != nil {
			log.Fatalf("Failed to chown %s: %v", target.path, err)
		}
		log.Printf("chowned %s to %d:%d", target.path, target.uid, target.gid)
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

// describeProcess reports the identity and effective capabilities of this
// process so init container logs show what the chown was allowed to do. It
// returns an empty string where /proc is not available.
func describeProcess() string {
	f, err := os.Open("/proc/self/status")
	if err != nil {
		return ""
	}
	defer f.Close()

	var fields []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		for _, key := range []string{"Uid:", "Gid:", "Groups:", "CapEff:"} {
			if strings.HasPrefix(line, key) {
				fields = append(fields, strings.TrimSuffix(key, ":")+"="+strings.Join(strings.Fields(line[len(key):]), ","))
			}
		}
	}
	return strings.Join(fields, " ")
}
