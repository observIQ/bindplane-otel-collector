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
// CAP_DAC_READ_SEARCH. Seeding happens before the chown, so the files written
// here end up owned by the collector rather than by root.
package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
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

// maxID is the reserved id the kernel reads as "leave unchanged", so it is
// the exclusive upper bound for -uid and -gid.
const maxID uint = 0xFFFFFFFF

// options is the parsed command line.
type options struct {
	configPath  string
	loggingPath string
	overwrite   bool
	chownPath   string
	uid, gid    uint32
}

func main() {
	opts, err := parseArgs(os.Args[1:], os.Stderr)
	if err != nil {
		if errors.Is(err, flag.ErrHelp) {
			os.Exit(0)
		}
		os.Exit(2)
	}

	if desc := describeProcess(); desc != "" {
		log.Printf("running as %s", desc)
	}

	if err := run(opts, newFS()); err != nil {
		log.Fatalf("Failed to initialize container: %v", err)
	}
}

// parseArgs parses and validates the command line. Validation failures are
// reported to output together with the usage text.
func parseArgs(args []string, output io.Writer) (*options, error) {
	fset := flag.NewFlagSet("container-init", flag.ContinueOnError)
	fset.SetOutput(output)
	opts := &options{}
	fset.StringVar(&opts.configPath, "config", "", "absolute path to write the default collector config (required)")
	fset.StringVar(&opts.loggingPath, "logging", "", "absolute path to write the default logging config (required)")
	fset.BoolVar(&opts.overwrite, "overwrite", false, "overwrite existing files")
	fset.StringVar(&opts.chownPath, "chown", "", "absolute path to recursively chown to uid:gid after writing the files above, created if missing")
	uid := fset.Uint("uid", 0, "owner uid for -chown (required with -chown)")
	gid := fset.Uint("gid", 0, "owner gid for -chown (required with -chown)")
	if err := fset.Parse(args); err != nil {
		return nil, err
	}

	if err := validate(opts, *uid, *gid, fset.Args()); err != nil {
		fmt.Fprintf(output, "container-init: %v\n", err)
		fset.Usage()
		return nil, err
	}
	opts.uid = uint32(*uid) // #nosec G115 -- range checked by validate
	opts.gid = uint32(*gid) // #nosec G115 -- range checked by validate
	return opts, nil
}

func validate(opts *options, uid, gid uint, extra []string) error {
	if len(extra) > 0 {
		return fmt.Errorf("unexpected arguments: %v", extra)
	}
	for _, f := range []struct{ name, path string }{
		{"-config", opts.configPath},
		{"-logging", opts.loggingPath},
	} {
		if f.path == "" {
			return fmt.Errorf("%s is required", f.name)
		}
		if !filepath.IsAbs(f.path) {
			return fmt.Errorf("%s path %q must be absolute", f.name, f.path)
		}
	}

	if opts.chownPath == "" {
		if uid != 0 || gid != 0 {
			return errors.New("-uid and -gid are only valid together with -chown")
		}
		return nil
	}
	if !filepath.IsAbs(opts.chownPath) || filepath.Clean(opts.chownPath) != opts.chownPath || opts.chownPath == "/" {
		return fmt.Errorf("-chown path %q must be an absolute, clean path other than /", opts.chownPath)
	}
	for _, f := range []struct {
		name string
		id   uint
	}{{"-uid", uid}, {"-gid", gid}} {
		if f.id == 0 || f.id >= maxID {
			return fmt.Errorf("%s must be between 1 and %d", f.name, maxID-1)
		}
	}
	return nil
}

// run seeds the config and logging files and then, when -chown was given,
// hands the whole tree to uid:gid. The order matters: the files are written
// 0600 by whichever user runs this, so the chown is what makes them readable
// by the unprivileged collector.
func run(opts *options, ops fsOps) error {
	if err := seed(opts); err != nil {
		return err
	}
	if opts.chownPath == "" {
		return nil
	}
	st, err := chownTree(opts.chownPath, opts.uid, opts.gid, ops)
	log.Printf("chown %s to %d:%d: %s", opts.chownPath, opts.uid, opts.gid, st)
	if err != nil {
		return fmt.Errorf("chown %s: %w", opts.chownPath, err)
	}
	return nil
}

// seed creates the parent directories of the config and logging paths and
// writes the default file contents to them. Existing files are left untouched
// unless overwrite is set, so an agent keeps its OpAMP persisted configuration
// across restarts.
func seed(opts *options) error {
	files := []struct{ path, contents string }{
		{opts.configPath, defaultCollectorConfig},
		{opts.loggingPath, defaultLoggingConfig},
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
		if !opts.overwrite {
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
