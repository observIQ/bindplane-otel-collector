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

package keyvalue

import (
	"context"
	"fmt"
	"strings"

	"github.com/open-telemetry/opentelemetry-log-collection/entry"
	"github.com/open-telemetry/opentelemetry-log-collection/operator"
	"github.com/open-telemetry/opentelemetry-log-collection/operator/helper"

	"github.com/hashicorp/go-multierror"
)

func init() {
	operator.Register("key_value_parser", func() operator.Builder { return NewKVParserConfig("") })
}

// NewKVParserConfig creates a new key value parser config with default values
func NewKVParserConfig(operatorID string) *KVParserConfig {
	return &KVParserConfig{
		ParserConfig: helper.NewParserConfig(operatorID, "key_value_parser"),
		Delimiter:    "=",
	}
}

// KVParserConfig is the configuration of a key value parser operator.
type KVParserConfig struct {
	helper.ParserConfig `yaml:",inline"`

	Delimiter string `json:"delimiter" yaml:"delimiter"`
}

// Build will build a key value parser operator.
func (c KVParserConfig) Build(context operator.BuildContext) ([]operator.Operator, error) {
	parserOperator, err := c.ParserConfig.Build(context)
	if err != nil {
		return nil, err
	}

	if len(c.Delimiter) == 0 {
		return nil, fmt.Errorf("delimiter is a required parameter")
	}

	kvParser := &KVParser{
		ParserOperator: parserOperator,
		delimiter:      c.Delimiter,
	}

	return []operator.Operator{kvParser}, nil
}

// KVParser is an operator that parses key value pairs.
type KVParser struct {
	helper.ParserOperator
	delimiter string
}

// Process will parse an entry for key value pairs.
func (kv *KVParser) Process(ctx context.Context, entry *entry.Entry) error {
	return kv.ParserOperator.ProcessWith(ctx, entry, kv.parse)
}

// parse will parse a value as key values.
func (kv *KVParser) parse(value interface{}) (interface{}, error) {
	switch m := value.(type) {
	case string:
		return kv.parser(m, kv.delimiter)
	case []byte:
		return kv.parser(string(m), kv.delimiter)
	default:
		return nil, fmt.Errorf("type %T cannot be parsed as key value pairs", value)
	}
}

func (kv *KVParser) parser(input string, delimiter string) (map[string]interface{}, error) {
	if len(input) == 0 {
		return nil, fmt.Errorf("parse from field %s is empty", kv.ParseFrom.String())
	}

	parsed := make(map[string]interface{})

	var err error
	for _, raw := range splitStringByWhitespace(input) {
		m := strings.Split(raw, delimiter)
		if len(m) != 2 {
			e := fmt.Errorf("expected '%s' to split by '%s' into two items, got %d", raw, delimiter, len(m))
			err = multierror.Append(err, e)
			continue
		}

		key := cleanString(m[0])
		value := cleanString(m[1])

		// TODO: Check if key already exists and fail if so?
		parsed[key] = value
	}

	return parsed, err
}

// split on whitespace and preserve quoted text
func splitStringByWhitespace(input string) []string {
	quoted := false
	raw := strings.FieldsFunc(input, func(r rune) bool {
		if r == '"' {
			quoted = !quoted
		}
		return !quoted && r == ' '
	})
	return raw
}

// trim leading and trailing space
func cleanString(input string) string {
	if len(input) > 0 && input[0] == '"' {
		input = input[1:]
	}
	if len(input) > 0 && input[len(input)-1] == '"' {
		input = input[:len(input)-1]
	}
	return strings.TrimSpace(input)
}
