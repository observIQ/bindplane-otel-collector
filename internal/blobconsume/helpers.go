// Copyright observIQ, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package blobconsume //import "github.com/observiq/bindplane-otel-collector/internal/blobconsume"

import (
	"bytes"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"
	"time"

	"go.opentelemetry.io/collector/pipeline"
)

// TimeFormat is the format for the starting and end time
const TimeFormat = "2006-01-02T15:04"

// ErrInvalidEntityPath is the error for invalid entity path
var ErrInvalidEntityPath = errors.New("invalid entity path")

// strings that indicate what type of telemetry is in a entity
const (
	metricEntitySignifier = "metrics_"
	logsEntitySignifier   = "logs_"
	tracesEntitySignifier = "traces_"
)

// entityNameRegex is the regex used to detect if an entity matches the expected path
var entityNameRegex = regexp.MustCompile(`^(?:[^/]*/)?year=(\d{4})/month=(\d{2})/day=(\d{2})/hour=(\d{2})/(?:minute=(\d{2})/)?([^/].*)$`)

// ParseEntityPath returns true if the entity is within the existing time range
func ParseEntityPath(entityName string) (entityTime *time.Time, telemetryType pipeline.Signal, err error) {
	matches := entityNameRegex.FindStringSubmatch(entityName)
	if matches == nil {
		err = ErrInvalidEntityPath
		return
	}

	year := matches[1]
	month := matches[2]
	day := matches[3]
	hour := matches[4]

	minute := "00"
	if matches[5] != "" {
		minute = matches[5]
	}

	lastPart := matches[6]

	timeString := fmt.Sprintf("%s-%s-%sT%s:%s", year, month, day, hour, minute)

	// Parse the expected format
	parsedTime, timeErr := time.Parse(TimeFormat, timeString)
	if timeErr != nil {
		err = fmt.Errorf("parse entity time: %w", timeErr)
		return
	}
	entityTime = &parsedTime

	switch {
	case strings.Contains(lastPart, metricEntitySignifier):
		telemetryType = pipeline.SignalMetrics
	case strings.Contains(lastPart, logsEntitySignifier):
		telemetryType = pipeline.SignalLogs
	case strings.Contains(lastPart, tracesEntitySignifier):
		telemetryType = pipeline.SignalTraces
	}

	return
}

// IsInTimeRange returns true if startingTime <= entityTime <= endingTime
func IsInTimeRange(entityTime, startingTime, endingTime time.Time) bool {
	return (entityTime.Equal(startingTime) || entityTime.After(startingTime)) &&
		(entityTime.Equal(endingTime) || entityTime.Before(endingTime))
}

// GzipDecompress does a gzip decompression on the passed in contents
func GzipDecompress(contents []byte) ([]byte, error) {
	gr, err := gzip.NewReader(bytes.NewBuffer(contents))
	if err != nil {
		return nil, fmt.Errorf("new reader: %w", err)
	}

	result, err := io.ReadAll(gr)
	if err != nil {
		return nil, fmt.Errorf("decompression: %w", err)
	}

	if err := gr.Close(); err != nil {
		return nil, fmt.Errorf("reader close: %w", err)
	}

	return result, nil
}
