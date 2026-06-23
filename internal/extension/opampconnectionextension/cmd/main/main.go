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

// Package main is the entry point for the ocb-built BDOT Collector. The
// `agent` Make target copies this file over ocb's generated main.go after
// `builder --skip-compilation` runs; `go build` then compiles it together
// with ocb's components.go inside ./build/.
//
// In its source location this file compiles against the stub `components()`
// in components_stub.go (`go test ./...` inside the extension module is
// fine). When it lands in ./build/, the stub is not copied — ocb's
// generated components.go is the only definition of `components()` and
// returns the manifest's full factory set.
//
// Everything else — flag parsing, managed/standalone dispatch, OpAMP
// wiring, collector lifecycle — lives behind
// opampconnectionextension/runtime.Run.
package main

import (
	"fmt"
	"log"
	"os"
	"strings"
	_ "time/tzdata"

	"github.com/observiq/bindplane-otel-collector/internal/extension/opampconnectionextension/runtime"
	"github.com/observiq/bindplane-otel-contrib/pkg/version"
	"github.com/spf13/pflag"
)

// Env-var fallbacks for paths (matches legacy cmd/collector behavior).
const (
	configPathENV   = "CONFIG_YAML_PATH"
	managerPathENV  = "MANAGER_YAML_PATH"
	loggingPathENV  = "LOGGING_YAML_PATH"
	featureGatesENV = "COLLECTOR_FEATURE_GATES"
)

func main() {
	collectorConfigPaths := pflag.StringSlice("config", defaultCollectorPaths(), "the collector config path")
	managerConfigPath := pflag.String("manager", defaultManagerPath(), "The configuration for remote management")
	loggingConfigPath := pflag.String("logging", defaultLoggingPath(), "the collector logging config path")
	featureGates := pflag.StringSlice("feature-gates", defaultFeatureGates(), "the collector feature gates")

	_ = pflag.String("log-level", "", "not implemented") // TEMP(jsirianni): Required for OTEL k8s operator
	var showVersion = pflag.BoolP("version", "v", false, "prints the version of the collector")
	pflag.Parse()

	if *showVersion {
		fmt.Println("observiq-otel-collector version", version.Version())
		fmt.Println("commit:", version.GitHash())
		fmt.Println("built at:", version.Date())
		return
	}

	factories, err := components()
	if err != nil {
		log.Fatalf("Failed to build factories from manifest: %v", err)
	}

	runtime.Run(runtime.Options{
		Factories:            factories,
		Version:              version.Version(),
		CollectorConfigPaths: *collectorConfigPaths,
		ManagerConfigPath:    *managerConfigPath,
		LoggingConfigPath:    *loggingConfigPath,
		FeatureGates:         *featureGates,
	})
}

func defaultCollectorPaths() []string {
	if cp, ok := os.LookupEnv(configPathENV); ok {
		return []string{cp}
	}
	return []string{"./config.yaml"}
}

func defaultManagerPath() string {
	if mp, ok := os.LookupEnv(managerPathENV); ok {
		return mp
	}
	return "./manager.yaml"
}

func defaultLoggingPath() string {
	if lp, ok := os.LookupEnv(loggingPathENV); ok {
		return lp
	}
	return runtime.DefaultLoggingPath
}

func defaultFeatureGates() []string {
	if fg, ok := os.LookupEnv(featureGatesENV); ok {
		return strings.Split(fg, ",")
	}
	return nil
}
