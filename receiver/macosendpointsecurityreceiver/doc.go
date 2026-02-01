// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

//go:generate mdatagen metadata.yaml

// Package macosendpointsecurityreceiver implements a receiver that uses the native
// macOS `eslogger` command to retrieve and parse Endpoint Security events.
// It streams Endpoint Security events in real-time using JSON format.
package macosendpointsecurityreceiver // import "github.com/observiq/bindplane-otel-collector/receiver/macosendpointsecurityreceiver"
