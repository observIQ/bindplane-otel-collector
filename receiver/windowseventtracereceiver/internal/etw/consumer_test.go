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

//go:build windows

package etw

import (
	"encoding/xml"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/observiq/bindplane-otel-collector/receiver/windowseventtracereceiver/internal/etw/advapi32"
)

// TestXMLEscape verifies that xmlEscape correctly escapes XML special characters
// so that dynamic ETW values cannot inject malformed XML.
func TestXMLEscape(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "no special characters",
			input:    "Microsoft-Windows-Security-Auditing",
			expected: "Microsoft-Windows-Security-Auditing",
		},
		{
			name:     "ampersand",
			input:    "Fonts & Colors",
			expected: "Fonts &amp; Colors",
		},
		{
			name:     "less than",
			input:    "size < 100",
			expected: "size &lt; 100",
		},
		{
			name:     "greater than",
			input:    "size > 100",
			expected: "size &gt; 100",
		},
		{
			name:     "double quote in attribute context",
			input:    `say "hello"`,
			expected: "say &#34;hello&#34;",
		},
		{
			name:     "single quote",
			input:    "it's here",
			expected: "it&#39;s here",
		},
		{
			name:     "command line with injection attempt",
			input:    `cmd.exe /c echo </EventData><Injected/>`,
			expected: `cmd.exe /c echo &lt;/EventData&gt;&lt;Injected/&gt;`,
		},
		{
			name:     "XML payload in value",
			input:    `<script>alert('xss')</script>`,
			expected: `&lt;script&gt;alert(&#39;xss&#39;)&lt;/script&gt;`,
		},
		{
			name:     "file path with ampersand",
			input:    `C:\Program Files (x86)\AT&T\tool.exe`,
			expected: `C:\Program Files (x86)\AT&amp;T\tool.exe`,
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "already safe characters",
			input:    "ProcessId-123_foo.bar",
			expected: "ProcessId-123_foo.bar",
		},
		{
			name:     "multiple special characters",
			input:    `a & b < c > d "e" 'f'`,
			expected: `a &amp; b &lt; c &gt; d &#34;e&#34; &#39;f&#39;`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, xmlEscape(tt.input))
		})
	}
}

// TestXMLEscape_RoundTrip verifies that xmlEscape output is valid XML by
// parsing it back and recovering the original string.
func TestXMLEscape_RoundTrip(t *testing.T) {
	inputs := []string{
		`cmd.exe /c del /f "C:\important & critical\file.txt"`,
		`<EventData><Data Name='injected'>payload</Data></EventData>`,
		"AT&T wireless <signal> 'good'",
		"normal provider name",
		"",
	}

	for _, input := range inputs {
		t.Run(input, func(t *testing.T) {
			escaped := xmlEscape(input)
			doc := "<root>" + escaped + "</root>"
			var result struct {
				Value string `xml:",chardata"`
			}
			err := xml.Unmarshal([]byte(doc), &result)
			require.NoError(t, err, "escaped value produced invalid XML for input %q", input)
			assert.Equal(t, input, result.Value, "round-trip should recover the original string")
		})
	}
}

// TestRawEventCallback_XMLEscaping verifies that property values containing XML
// special characters are escaped in the raw event output, preventing injection.
func TestRawEventCallback_XMLEscaping(t *testing.T) {
	tests := []struct {
		name          string
		eventData     map[string]any
		checkContains []string
		checkAbsent   []string
	}{
		{
			name:      "plain value is preserved",
			eventData: map[string]any{"Message": "hello world"},
			checkContains: []string{
				`<Data Name="Message">hello world</Data>`,
			},
		},
		{
			name:      "XML injection in value is escaped",
			eventData: map[string]any{"CommandLine": `</EventData><Injected/>`},
			checkContains: []string{
				`<Data Name="CommandLine">&lt;/EventData&gt;&lt;Injected/&gt;</Data>`,
			},
			checkAbsent: []string{
				`</EventData><Injected/>`,
			},
		},
		{
			name:      "ampersand in value is escaped",
			eventData: map[string]any{"Company": `AT&T`},
			checkContains: []string{
				`<Data Name="Company">AT&amp;T</Data>`,
			},
			checkAbsent: []string{"AT&T</Data>"},
		},
		{
			name:      "quotes in value are escaped",
			eventData: map[string]any{"CommandLine": `cmd.exe /c echo "hello"`},
			checkContains: []string{
				`<Data Name="CommandLine">cmd.exe /c echo &#34;hello&#34;</Data>`,
			},
		},
		{
			name:      "special characters in property name are escaped",
			eventData: map[string]any{`Key<Inject>`: "value"},
			checkContains: []string{
				`<Data Name="Key&lt;Inject&gt;">value</Data>`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			consumer := newTestConsumer()
			consumer.getEventProperties = func(_ *advapi32.EventRecord, _ *zap.Logger) (map[string]any, error) {
				return tt.eventData, nil
			}

			record := &advapi32.EventRecord{}
			rc := consumer.rawEventCallback(record)
			assert.Equal(t, uintptr(0), rc)

			event := <-consumer.Events
			require.NotNil(t, event)

			for _, want := range tt.checkContains {
				assert.Contains(t, event.Raw, want)
			}
			for _, absent := range tt.checkAbsent {
				assert.NotContains(t, event.Raw, absent)
			}

			// Every output must parse as valid XML.
			err := xml.Unmarshal([]byte(event.Raw), new(any))
			assert.NoError(t, err, "rawEventCallback output must be valid XML:\n%s", event.Raw)
		})
	}
}

// TestRawEventCallback_XMLStructure verifies that the System section fields
// derived from EventHeader are correctly rendered.
func TestRawEventCallback_XMLStructure(t *testing.T) {
	consumer := newTestConsumer()
	consumer.getEventProperties = func(_ *advapi32.EventRecord, _ *zap.Logger) (map[string]any, error) {
		return map[string]any{"Prop": "val"}, nil
	}

	record := &advapi32.EventRecord{}
	record.EventHeader.EventDescriptor.Level = 3
	record.EventHeader.EventDescriptor.Version = 2
	record.EventHeader.ProcessId = 1234
	record.EventHeader.ThreadId = 5678

	rc := consumer.rawEventCallback(record)
	assert.Equal(t, uintptr(0), rc)

	event := <-consumer.Events
	require.NotNil(t, event)

	assert.Contains(t, event.Raw, "<Event>")
	assert.Contains(t, event.Raw, "</Event>")
	assert.Contains(t, event.Raw, "<System>")
	assert.Contains(t, event.Raw, "</System>")
	assert.Contains(t, event.Raw, "<EventData>")
	assert.Contains(t, event.Raw, "</EventData>")
	assert.Contains(t, event.Raw, "<Level>3</Level>")
	assert.Contains(t, event.Raw, "<Version>2</Version>")
	assert.Contains(t, event.Raw, `ProcessID="1234"`)
	assert.Contains(t, event.Raw, `ThreadID="5678"`)

	err := xml.Unmarshal([]byte(event.Raw), new(any))
	assert.NoError(t, err, "output must be valid XML:\n%s", event.Raw)
}

// newTestConsumer returns a minimal Consumer for unit tests.
// getEventProperties must be set before calling rawEventCallback.
func newTestConsumer() *Consumer {
	return &Consumer{
		Events:      make(chan *Event, 1),
		doneChan:    make(chan struct{}),
		wg:          &sync.WaitGroup{},
		logger:      zap.NewNop(),
		providerMap: map[string]*Provider{},
	}
}
