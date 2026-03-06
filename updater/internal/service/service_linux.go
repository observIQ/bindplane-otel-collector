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

//go:build linux

package service

import (
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/observiq/bindplane-otel-collector/updater/internal/file"
	"github.com/observiq/bindplane-otel-collector/updater/internal/path"
	"go.uber.org/zap"
)

// needsSudo returns true if the current process is not running as root.
func needsSudo() bool {
	return os.Getuid() != 0
}

// sudoCommand creates an exec.Cmd, prepending "sudo" if the process is non-root.
func sudoCommand(name string, args ...string) *exec.Cmd {
	if needsSudo() {
		allArgs := append([]string{name}, args...)
		//#nosec G204 -- arguments are not user-controlled
		return exec.Command("sudo", allArgs...)
	}
	//#nosec G204 -- arguments are not user-controlled
	return exec.Command(name, args...)
}

// Option is an extra option for creating a Service
type Option func(linuxSvc linuxService)

//type Option func(linuxSvc *linuxSysVService)

// WithServiceFile returns an option setting the service file to use when updating using the service
func WithServiceFile(svcFilePath string) Option {
	return func(linuxSvc linuxService) {
		linuxSvc.setNewSvcFile(svcFilePath)
	}
}

// NewService returns an instance of the Service interface for managing the observiq-otel-collector service on the current OS.
func NewService(logger *zap.Logger, installDir string, opts ...Option) Service {
	// Get some information from the environment
	_, svcFileName := filepath.Split(path.LinuxServiceFilePath())

	var linuxSvc linuxService

	// Base struct choice on
	if path.LinuxServiceCmdName() == "systemctl" {
		linuxSvc = &linuxSystemdService{
			newServiceFilePath:       filepath.Join(path.ServiceFileDir(installDir), svcFileName),
			serviceName:              svcFileName,
			installedServiceFilePath: path.SystemdFilePath,
			installDir:               installDir,
			logger:                   logger.Named("linux-systemd-service"),
		}
	} else {
		linuxSvc = &linuxSysVService{
			newServiceFilePath:       filepath.Join(path.ServiceFileDir(installDir), svcFileName),
			serviceName:              svcFileName,
			installedServiceFilePath: path.SysVFilePath,
			installDir:               installDir,
			logger:                   logger.Named("linux-sysv-service"),
		}
	}

	for _, opt := range opts {
		opt(linuxSvc)
	}

	return linuxSvc
}

type linuxService interface {
	Service
	setNewSvcFile(string)
}

type linuxSystemdService struct {
	// newServiceFilePath is the file path to the new unit file
	newServiceFilePath string
	// serviceName is the name of the service
	serviceName string
	// installedServiceFilePath is the file path to the installed unit file
	installedServiceFilePath string
	installDir               string
	logger                   *zap.Logger
}

// Start the service
func (l linuxSystemdService) Start() error {
	cmd := sudoCommand("systemctl", "start", l.serviceName)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("running systemctl failed: %w", err)
	}
	return nil
}

// Stop the service
func (l linuxSystemdService) Stop() error {
	cmd := sudoCommand("systemctl", "stop", l.serviceName)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("running systemctl failed: %w", err)
	}
	return nil
}

// installs the service
func (l linuxSystemdService) install() error {
	inFile, err := os.Open(l.newServiceFilePath)
	if err != nil {
		return fmt.Errorf("failed to open input file: %w", err)
	}
	defer func() {
		err := inFile.Close()
		if err != nil {
			log.Default().Printf("Service Install: Failed to close input file: %s", err)
		}
	}()

	outFile, err := os.OpenFile(l.installedServiceFilePath, os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return fmt.Errorf("failed to open output file: %w", err)
	}
	defer func() {
		err := outFile.Close()
		if err != nil {
			log.Default().Printf("Service Install: Failed to close output file: %s", err)
		}
	}()

	if _, err := io.Copy(outFile, inFile); err != nil {
		return fmt.Errorf("failed to copy service file: %w", err)
	}

	cmd := sudoCommand("systemctl", "daemon-reload")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("reloading systemctl failed: %w", err)
	}

	cmd = sudoCommand("systemctl", "enable", l.serviceName)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("enabling unit file failed: %w", err)
	}

	return nil
}

// uninstalls the service
func (l linuxSystemdService) uninstall() error {
	cmd := sudoCommand("systemctl", "disable", l.serviceName)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to disable unit: %w", err)
	}

	if err := os.Remove(l.installedServiceFilePath); err != nil {
		return fmt.Errorf("failed to remove service file: %w", err)
	}

	cmd = sudoCommand("systemctl", "daemon-reload")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("reloading systemctl failed: %w", err)
	}

	return nil
}

func (l linuxSystemdService) Update() error {
	if err := l.uninstall(); err != nil {
		return fmt.Errorf("failed to uninstall old service: %w", err)
	}

	if err := l.install(); err != nil {
		return fmt.Errorf("failed to install new service: %w", err)
	}

	return nil
}

func (l linuxSystemdService) Backup() error {
	if err := file.CopyFileNoOverwrite(l.logger.Named("copy-file"), l.installedServiceFilePath, path.BackupServiceFile(l.installDir)); err != nil {
		return fmt.Errorf("failed to copy service file: %w", err)
	}

	return nil
}

func (l *linuxSystemdService) setNewSvcFile(newFilePath string) {
	l.newServiceFilePath = newFilePath
}

type linuxSysVService struct {
	// newServiceFilePath is the file path to the new unit file
	newServiceFilePath string
	// serviceName is the name of the service
	serviceName string
	// installedServiceFilePath is the file path to the installed unit file
	installedServiceFilePath string
	installDir               string
	logger                   *zap.Logger
}

// Start the service
func (l linuxSysVService) Start() error {
	cmd := sudoCommand("service", l.serviceName, "start")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("running service failed: %w", err)
	}
	return nil
}

// Stop the service
func (l linuxSysVService) Stop() error {
	cmd := sudoCommand("service", l.serviceName, "stop")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("running service failed: %w", err)
	}
	return nil
}

// installs the service
func (l linuxSysVService) install() error {
	inFile, err := os.Open(l.newServiceFilePath)
	if err != nil {
		return fmt.Errorf("failed to open input file: %w", err)
	}
	defer func() {
		err := inFile.Close()
		if err != nil {
			log.Default().Printf("Service Install: Failed to close input file: %s", err)
		}
	}()

	//#nosec G302 -- File permissions for the service file match install script and other installed services
	outFile, err := os.OpenFile(l.installedServiceFilePath, os.O_CREATE|os.O_WRONLY, 0755)
	if err != nil {
		return fmt.Errorf("failed to open output file: %w", err)
	}
	defer func() {
		err := outFile.Close()
		if err != nil {
			log.Default().Printf("Service Install: Failed to close output file: %s", err)
		}
	}()

	if _, err := io.Copy(outFile, inFile); err != nil {
		return fmt.Errorf("failed to copy service file: %w", err)
	}

	cmd := sudoCommand("chkconfig", l.serviceName, "on")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("chkconfig on failed: %w", err)
	}

	return nil
}

// uninstalls the service
func (l linuxSysVService) uninstall() error {
	cmd := sudoCommand("chkconfig", l.serviceName, "off")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("chkconfig off failed: %w", err)
	}

	if err := os.Remove(l.installedServiceFilePath); err != nil {
		return fmt.Errorf("failed to remove service file: %w", err)
	}

	return nil
}

func (l linuxSysVService) Update() error {
	if err := l.uninstall(); err != nil {
		return fmt.Errorf("failed to uninstall old service: %w", err)
	}

	if err := l.install(); err != nil {
		return fmt.Errorf("failed to install new service: %w", err)
	}

	return nil
}

func (l linuxSysVService) Backup() error {
	if err := file.CopyFileNoOverwrite(l.logger.Named("copy-file"), l.installedServiceFilePath, path.BackupServiceFile(l.installDir)); err != nil {
		return fmt.Errorf("failed to copy service file: %w", err)
	}

	return nil
}

func (l *linuxSysVService) setNewSvcFile(newFilePath string) {
	l.newServiceFilePath = newFilePath
}
