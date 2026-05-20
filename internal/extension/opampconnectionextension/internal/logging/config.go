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

// Package logging parses and applies logging configuration
package logging

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
	"gopkg.in/yaml.v3"
)

// DefaultConfigPath is the relative path to the default logging.yaml
const DefaultConfigPath = "./logging.yaml"

const (
	// fileOutput is an output option for logging to a file.
	fileOutput string = "file"

	// stdOutput is an output option for logging to stdout.
	stdOutput string = "stdout"
)

// LoggerConfig is the configuration of a logger.
type LoggerConfig struct {
	Output string             `yaml:"output"`
	Level  zapcore.Level      `yaml:"level"`
	File   *lumberjack.Logger `yaml:"file,omitempty"`
}

// NewLoggerConfig returns a logger config.
// If configPath is not set, stdout logging will be enabled, and a default
// configuration will be written to ./logging.yaml
func NewLoggerConfig(configPath string) (*LoggerConfig, error) {
	// No logger path specified, we'll assume the default path.
	if configPath == DefaultConfigPath {
		// If the file doesn't exist, we will create the config with the default parameters.
		if _, err := os.Stat(DefaultConfigPath); errors.Is(err, os.ErrNotExist) {
			defaultConf := defaultConfig()
			if err := writeConfig(defaultConf, DefaultConfigPath); err != nil {
				return nil, fmt.Errorf("failed to write default configuration: %w", err)
			}
			return defaultConf, nil
		} else if err != nil {
			return nil, fmt.Errorf("failed to stat config: %w", err)
		}
		// The config already exists; We should continue and read it like any other config.
	}

	cleanPath := filepath.Clean(configPath)

	// conf will start as the default config; any unspecified values in the config
	// will default to the values in the default config.
	conf := defaultConfig()

	confBytes, err := os.ReadFile(cleanPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	if err := yaml.Unmarshal(confBytes, conf); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	if conf.File != nil {
		// Expand optional environment variables in file path
		conf.File.Filename = os.ExpandEnv(conf.File.Filename)
	}

	return conf, nil
}

// Options returns the LoggerConfig's zap logging options.
func (l *LoggerConfig) Options() ([]zap.Option, error) {
	core, err := l.core()
	if err != nil {
		return nil, err
	}

	opt := zap.WrapCore(func(_ zapcore.Core) zapcore.Core {
		return core
	})

	return []zap.Option{opt}, nil
}

// core returns the logging core specified in the config.
// An unknown output will return a nop core.
func (l *LoggerConfig) core() (zapcore.Core, error) {
	switch l.Output {
	case fileOutput:
		return zapcore.NewCore(newEncoder(), zapcore.AddSync(l.File), l.Level), nil
	case stdOutput:
		return zapcore.NewCore(newEncoder(), zapcore.Lock(os.Stdout), l.Level), nil
	default:
		return nil, fmt.Errorf("unrecognized output type: %s", l.Output)
	}
}

func newEncoder() zapcore.Encoder {
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	return zapcore.NewJSONEncoder(encoderConfig)
}

// defaultConfig returns a new instance of the default logging configuration
func defaultConfig() *LoggerConfig {
	return &LoggerConfig{
		Output: stdOutput,
		Level:  zap.InfoLevel,
	}
}

// writeConfig writes the given configuration to the specified location.
func writeConfig(config *LoggerConfig, outLocation string) error {
	configBytes, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal: %w", err)
	}

	if err = os.WriteFile(outLocation, configBytes, 0600); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}
