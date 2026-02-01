// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

//go:build darwin

package macosendpointsecurityreceiver

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/confmap/confmaptest"
)

func TestDebugConfig(t *testing.T) {
	// Load the debug test config
	cm, err := confmaptest.LoadConf("test_config_debug.yaml")
	require.NoError(t, err)

	// Get the receiver config section
	receiverSub, err := cm.Sub("receivers::macosendpointsecurity")
	require.NoError(t, err)

	// Unmarshal into Config struct
	cfg := &Config{}
	err = receiverSub.Unmarshal(cfg)
	require.NoError(t, err)

	// Validate the config
	err = cfg.Validate()
	require.NoError(t, err)

	// Verify expected values
	require.Len(t, cfg.EventTypes, 3)
	require.Contains(t, cfg.EventTypes, EventTypeExec)
	require.Contains(t, cfg.EventTypes, EventTypeFork)
	require.Contains(t, cfg.EventTypes, EventTypeExit)
}
