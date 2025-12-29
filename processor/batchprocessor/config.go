// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package batchprocessor // import "go.opentelemetry.io/collector/processor/batchprocessor"

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"go.opentelemetry.io/collector/component"
)

// Config defines configuration for batch processor.
type Config struct {
	// Timeout sets the time after which a batch will be sent regardless of size.
	// When this is set to zero, batched data will be sent immediately.
	Timeout time.Duration `mapstructure:"timeout"`

	// SendBatchSize is the size of a batch which after hit, will trigger it to be sent.
	// When this is set to zero, the batch size is ignored and data will be sent immediately
	// subject to only send_batch_max_size.
	SendBatchSize uint32 `mapstructure:"send_batch_size"`

	// SendBatchMaxSize is the maximum size of a batch. It must be larger than SendBatchSize.
	// Larger batches are split into smaller units.
	// Default value is 0, that means no maximum size.
	SendBatchMaxSize uint32 `mapstructure:"send_batch_max_size"`

	// SendBatchSizeBytes is the size of a batch in bytes which after hit, will trigger it to be sent.
	// When this is set to zero, the batch size in bytes is ignored.
	// This works in conjunction with SendBatchSize - a batch is sent when either threshold is reached.
	SendBatchSizeBytes uint64 `mapstructure:"send_batch_size_bytes"`

	// SendBatchMaxSizeBytes is the maximum size of a batch in bytes. It must be larger than SendBatchSizeBytes.
	// Larger batches are split into smaller units.
	// Default value is 0, that means no maximum size in bytes.
	SendBatchMaxSizeBytes uint64 `mapstructure:"send_batch_max_size_bytes"`

	// MetadataKeys is a list of client.Metadata keys that will be
	// used to form distinct batchers.  If this setting is empty,
	// a single batcher instance will be used.  When this setting
	// is not empty, one batcher will be used per distinct
	// combination of values for the listed metadata keys.
	//
	// Empty value and unset metadata are treated as distinct cases.
	//
	// Entries are case-insensitive.  Duplicated entries will
	// trigger a validation error.
	MetadataKeys []string `mapstructure:"metadata_keys"`

	// MetadataCardinalityLimit indicates the maximum number of
	// batcher instances that will be created through a distinct
	// combination of MetadataKeys.
	MetadataCardinalityLimit uint32 `mapstructure:"metadata_cardinality_limit"`

	// BatchGroupByAttributes is a list of attribute keys that will be
	// used to form distinct batchers. If this setting is empty,
	// data will be batched in FIFO order. When this setting
	// is not empty, one batcher will be used per distinct
	// combination of values for the listed attribute keys.
	//
	// Attributes are extracted from the telemetry data itself:
	// - For traces: span attributes, resource attributes, or scope attributes
	// - For logs: log record attributes, resource attributes, or scope attributes
	// - For metrics: datapoint attributes, resource attributes, or scope attributes
	//
	// Empty value and missing attributes are treated as distinct cases.
	//
	// Entries are case-sensitive. Duplicated entries will
	// trigger a validation error.
	BatchGroupByAttributes []string `mapstructure:"batch_group_by_attributes"`

	// BatchGroupByAttributeLimit indicates the maximum number of
	// batcher instances that will be created through a distinct
	// combination of BatchGroupByAttributes.
	BatchGroupByAttributeLimit uint32 `mapstructure:"batch_group_by_attribute_limit"`
	// prevent unkeyed literal initialization
	_ struct{}
}

var _ component.Config = (*Config)(nil)

// Validate checks if the processor configuration is valid
func (cfg *Config) Validate() error {
	if cfg.SendBatchMaxSize > 0 && cfg.SendBatchMaxSize < cfg.SendBatchSize {
		return errors.New("send_batch_max_size must be greater or equal to send_batch_size")
	}
	if cfg.SendBatchMaxSizeBytes > 0 && cfg.SendBatchMaxSizeBytes < cfg.SendBatchSizeBytes {
		return errors.New("send_batch_max_size_bytes must be greater or equal to send_batch_size_bytes")
	}
	uniq := map[string]bool{}
	for _, k := range cfg.MetadataKeys {
		l := strings.ToLower(k)
		if _, has := uniq[l]; has {
			return fmt.Errorf("duplicate entry in metadata_keys: %q (case-insensitive)", l)
		}
		uniq[l] = true
	}
	if cfg.Timeout < 0 {
		return errors.New("timeout must be greater or equal to 0")
	}
	// Validate BatchGroupByAttributes for duplicates
	attrUniq := map[string]bool{}
	for _, k := range cfg.BatchGroupByAttributes {
		if _, has := attrUniq[k]; has {
			return fmt.Errorf("duplicate entry in batch_group_by_attributes: %q", k)
		}
		attrUniq[k] = true
	}
	// Ensure metadata_keys and batch_group_by_attributes are not both set
	if len(cfg.MetadataKeys) > 0 && len(cfg.BatchGroupByAttributes) > 0 {
		return errors.New("metadata_keys and batch_group_by_attributes cannot both be set")
	}
	return nil
}
