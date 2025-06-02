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

package webhookexporter

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHTTPVerb_UnmarshalText(t *testing.T) {
	tests := []struct {
		name    string
		input   []byte
		want    HTTPVerb
		wantErr bool
	}{
		{
			name:    "valid POST",
			input:   []byte("POST"),
			want:    POST,
			wantErr: false,
		},
		{
			name:    "valid PATCH",
			input:   []byte("PATCH"),
			want:    PATCH,
			wantErr: false,
		},
		{
			name:    "valid PUT",
			input:   []byte("PUT"),
			want:    PUT,
			wantErr: false,
		},
		{
			name:    "invalid verb",
			input:   []byte("GET"),
			want:    "",
			wantErr: true,
		},
		{
			name:    "empty verb",
			input:   []byte(""),
			want:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var v HTTPVerb
			err := v.unmarshalText(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Empty(t, v)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, v)
			}
		})
	}
}

func TestEndpoint_UnmarshalText(t *testing.T) {
	tests := []struct {
		name    string
		input   []byte
		want    Endpoint
		wantErr bool
	}{
		{
			name:    "valid http endpoint",
			input:   []byte("http://example.com"),
			want:    Endpoint("http://example.com"),
			wantErr: false,
		},
		{
			name:    "valid https endpoint",
			input:   []byte("https://example.com"),
			want:    Endpoint("https://example.com"),
			wantErr: false,
		},
		{
			name:    "invalid scheme",
			input:   []byte("ftp://example.com"),
			want:    "",
			wantErr: true,
		},
		{
			name:    "missing scheme",
			input:   []byte("example.com"),
			want:    "",
			wantErr: true,
		},
		{
			name:    "empty endpoint",
			input:   []byte(""),
			want:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var e Endpoint
			err := e.unmarshalText(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Empty(t, e)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, e)
			}
		})
	}
}

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
	}{
		{
			name: "valid config",
			config: Config{
				Endpoint:    "https://example.com",
				Verb:        POST,
				ContentType: "application/json",
			},
			wantErr: false,
		},
		{
			name: "invalid endpoint",
			config: Config{
				Endpoint:    "ftp://example.com",
				Verb:        POST,
				ContentType: "application/json",
			},
			wantErr: true,
		},
		{
			name: "invalid verb",
			config: Config{
				Endpoint:    "https://example.com",
				Verb:        "GET",
				ContentType: "application/json",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
