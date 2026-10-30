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

package service

import (
	"fmt"
	"os"
	"os/exec"
	"time"

	"go.uber.org/zap"
)

const (
	aixSubsystem = "observiq-otel-collector"
	aixGroup     = "oiqcollector"
	aixEnvFile   = "/etc/observiq-otel-collector.env"
)

// Option is an extra option for creating a Service
type Option func(aixSvc *aixService)

// WithServiceFile is a no-op on AIX, where SRC keeps the service definition instead of a file.
func WithServiceFile(_ string) Option {
	return func(_ *aixService) {}
}

// NewService returns an instance of the Service interface for managing the observiq-otel-collector service on the current OS.
func NewService(logger *zap.Logger, installDir string, opts ...Option) Service {
	aixSvc := &aixService{
		src:    srcService{subsystem: aixSubsystem, group: aixGroup, installDir: installDir, envFile: aixEnvFile},
		poller: newSRCPoller(aixSubsystem),
		logger: logger.Named("aix-service"),
	}

	for _, opt := range opts {
		opt(aixSvc)
	}

	return aixSvc
}

// newSRCPoller polls the named subsystem with lssrc on the real clock.
func newSRCPoller(subsystem string) srcPoller {
	return srcPoller{
		lssrc: func() ([]byte, error) {
			//#nosec G204 -- subsystem is a constant
			return exec.Command("lssrc", "-s", subsystem).Output()
		},
		now:   time.Now,
		sleep: time.Sleep,
	}
}

type aixService struct {
	src    srcService
	poller srcPoller
	logger *zap.Logger
}

// run runs an SRC command and includes its output in the error.
func (a aixService) run(name string, args ...string) error {
	//#nosec G204 -- arguments are built from constants and the install directory
	out, err := exec.Command(name, args...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("running %s failed: %w: %s", name, err, out)
	}
	return nil
}

// Start the service
func (a aixService) Start() error {
	env, err := os.ReadFile(a.src.envFile)
	if err != nil {
		a.logger.Warn("Starting without the env file", zap.String("path", a.src.envFile), zap.Error(err))
	}
	if err := a.run("startsrc", a.src.startsrcArgs(env)...); err != nil {
		return err
	}
	return a.poller.waitFor("active", srcWaitTimeout)
}

// Stop the service
func (a aixService) Stop() error {
	if err := a.run("stopsrc", "-g", a.src.group); err != nil {
		return err
	}
	return a.poller.waitFor("inoperative", srcWaitTimeout)
}

// install defines the subsystem and the inittab entry that starts it at boot
func (a aixService) install() error {
	if err := a.run("mkssys", a.src.mkssysArgs()...); err != nil {
		return err
	}
	return a.run("mkitab", a.src.inittabEntry())
}

// uninstall stops the subsystem if it's running, then removes it and its inittab entry
func (a aixService) uninstall() error {
	out, _ := a.poller.lssrc()
	if srcActive(out) {
		if err := a.Stop(); err != nil {
			return err
		}
	}
	if err := a.run("rmitab", a.src.group); err != nil {
		return err
	}
	return a.run("rmssys", "-s", a.src.subsystem)
}

// Update replaces the subsystem definition with the current one
func (a aixService) Update() error {
	if err := a.uninstall(); err != nil {
		return fmt.Errorf("failed to uninstall old service: %w", err)
	}

	if err := a.install(); err != nil {
		return fmt.Errorf("failed to install new service: %w", err)
	}

	return nil
}

// Backup is a no-op, since SRC has no service file to back up.
func (a aixService) Backup() error {
	return nil
}
