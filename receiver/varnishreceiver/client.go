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

package varnishreceiver // import "github.com/observiq/observiq-otel-collector/receiver/varnishreceiver"

import (
	"encoding/json"
	"os/exec"

	"go.opentelemetry.io/collector/component"
	"go.uber.org/zap"
)

// executer executes commands.
type executer interface {
	Execute(command string, args []string) ([]byte, error)
}

// varnishExecuter executes varnish commands.
type varnishExecuter struct{}

// newExecuter creates a new executer.
func newExecuter() executer {
	return &varnishExecuter{}
}

// Execute executes commands with args flag.
func (e *varnishExecuter) Execute(command string, args []string) ([]byte, error) {
	return exec.Command(command, args...).Output()
}

// client is an interface to get stats and build the exec command using an executer.
type client interface {
	GetStats() (*Stats, error)
}

var _ client = (*varnishClient)(nil)

type varnishClient struct {
	exec   executer
	cfg    *Config
	logger *zap.Logger
}

// newVarnishClient creates a client.
func newVarnishClient(cfg *Config, _ component.Host, settings component.TelemetrySettings) client {
	return &varnishClient{
		cfg:    cfg,
		logger: settings.Logger,
		exec:   newExecuter(),
	}
}

const (
	varnishStat = "varnishstat"
	counters    = "counters"
)

// BuildCommand builds the exec command statement.
func (v *varnishClient) BuildCommand() (string, []string) {
	argList := []string{"-j"}
	argList = append(argList, "-n", v.cfg.CacheDir)

	command := varnishStat
	if v.cfg.ExecDir != "" {
		command = v.cfg.ExecDir
	}
	return command, argList
}

// GetStats executes and parses the varnish stats.
func (v *varnishClient) GetStats() (*Stats, error) {
	command, argList := v.BuildCommand()

	output, err := v.exec.Execute(command, argList)
	if err != nil {
		return nil, err
	}

	return parseStats(output)
}

// parseStats parses varnishStats json response into a Stats struct.
func parseStats(rawStats []byte) (*Stats, error) {
	raw := make(map[string]interface{})
	if err := json.Unmarshal(rawStats, &raw); err != nil {
		return nil, err
	}

	// Varnish 6.5+ nests metrics inside a "counters" field.
	// https://varnish-cache.org/docs/6.5/whats-new/upgrading-6.5.html#varnishstat
	if _, ok := raw[counters]; ok {
		var jsonParsed FullStats
		if err := json.Unmarshal(rawStats, &jsonParsed); err != nil {
			return nil, err
		}

		return &jsonParsed.Counters, nil
	}

	var jsonParsed Stats
	if err := json.Unmarshal(rawStats, &jsonParsed); err != nil {
		return nil, err
	}

	return &jsonParsed, nil
}
