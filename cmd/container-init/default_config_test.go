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

//go:build !windows

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

const imageConfigPath = "/etc/otel/config.yaml"

// TestDefaultCollectorConfigMatchesDockerfiles catches drift between
// defaultCollectorConfig and the copy each container image bakes in with a
// COPY heredoc.
func TestDefaultCollectorConfigMatchesDockerfiles(t *testing.T) {
	for _, name := range []string{"Dockerfile.ubuntu-collector"} {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join("..", "..", "docker", name)
			contents, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read %s: %v", path, err)
			}
			if strings.Contains(string(contents), "release_deps/config.yaml") {
				t.Fatalf("%s references release_deps/config.yaml; the default config must come from a COPY heredoc", path)
			}
			got, err := copyHeredoc(string(contents), imageConfigPath)
			if err != nil {
				t.Fatalf("%s: %v", path, err)
			}
			if got != defaultCollectorConfig {
				t.Fatalf("%s writes a %s that differs from defaultCollectorConfig in main.go; change them together\n got: %q\nwant: %q",
					path, imageConfigPath, got, defaultCollectorConfig)
			}
		})
	}
}

// TestDefaultCollectorConfigHealthCheck pins the health_check wiring that
// Kubernetes probes need, and that self-telemetry stays off so health_check is
// the only port the default config opens.
func TestDefaultCollectorConfigHealthCheck(t *testing.T) {
	for _, want := range []string{
		"extensions:\n  health_check:\n    endpoint: 0.0.0.0:13133\n",
		"service:\n  extensions: [health_check]\n",
		"  telemetry:\n    metrics:\n      level: none\n",
	} {
		if !strings.Contains(defaultCollectorConfig, want) {
			t.Errorf("defaultCollectorConfig lacks %q", want)
		}
	}
}

func TestCopyHeredoc(t *testing.T) {
	tests := []struct {
		name, dockerfile, want, wantErr string
	}{
		{
			name:       "double quoted",
			dockerfile: "FROM scratch\nCOPY <<\"EOF\" /etc/otel/config.yaml\na:\n  b: $HOME\nEOF\nUSER otel\n",
			want:       "a:\n  b: $HOME\n",
		},
		{
			name:       "single quoted with flags",
			dockerfile: "COPY --chmod=0600 <<'END' /etc/otel/config.yaml\na: 1\nEND\n",
			want:       "a: 1\n",
		},
		{
			name:       "other destinations ignored",
			dockerfile: "COPY <<\"EOF\" /etc/otel/other.yaml\nx: 1\nEOF\nCOPY <<\"EOF\" /etc/otel/config.yaml\na: 1\nEOF\n",
			want:       "a: 1\n",
		},
		{
			name:       "tab stripping",
			dockerfile: "COPY <<-\"EOF\" /etc/otel/config.yaml\na: 1\nEOF\n",
			wantErr:    "strips leading tabs",
		},
		{
			name:       "unquoted",
			dockerfile: "COPY <<EOF /etc/otel/config.yaml\na: 1\nEOF\n",
			wantErr:    "must be quoted",
		},
		{
			name:       "not a heredoc",
			dockerfile: "COPY release_deps/config.yaml /etc/otel/config.yaml\n",
			wantErr:    "not a heredoc",
		},
		{
			name:       "unterminated",
			dockerfile: "COPY <<\"EOF\" /etc/otel/config.yaml\na: 1\n",
			wantErr:    "not terminated",
		},
		{
			name:       "missing",
			dockerfile: "FROM scratch\n",
			wantErr:    "no COPY heredoc",
		},
		{
			name:       "duplicate",
			dockerfile: "COPY <<\"EOF\" /etc/otel/config.yaml\na: 1\nEOF\nCOPY <<\"EOF\" /etc/otel/config.yaml\na: 2\nEOF\n",
			wantErr:    "2 COPY heredocs",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := copyHeredoc(tt.dockerfile, imageConfigPath)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("err = %v, want it to contain %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("copyHeredoc: %v", err)
			}
			if got != tt.want {
				t.Fatalf("got %q, want %q", got, tt.want)
			}
		})
	}
}

// copyHeredoc returns the body of the single COPY heredoc in dockerfile that
// writes dest. The delimiter must be quoted so BuildKit does not expand
// variables in the body, and <<- is rejected because it strips leading tabs.
func copyHeredoc(dockerfile, dest string) (string, error) {
	lines := strings.Split(dockerfile, "\n")
	var bodies []string
	for i := 0; i < len(lines); i++ {
		fields := strings.Fields(lines[i])
		if len(fields) < 3 || !strings.EqualFold(fields[0], "COPY") || fields[len(fields)-1] != dest {
			continue
		}
		src := fields[len(fields)-2]
		if !strings.HasPrefix(src, "<<") {
			return "", fmt.Errorf("line %d: COPY to %s is not a heredoc", i+1, dest)
		}
		delim := strings.TrimPrefix(src, "<<")
		if strings.HasPrefix(delim, "-") {
			return "", fmt.Errorf("line %d: %s strips leading tabs; use <<", i+1, src)
		}
		if len(delim) < 3 || (delim[0] != '"' && delim[0] != '\'') || delim[len(delim)-1] != delim[0] {
			return "", fmt.Errorf("line %d: heredoc delimiter in %s must be quoted to disable variable expansion", i+1, src)
		}
		delim = delim[1 : len(delim)-1]
		end := slices.Index(lines[i+1:], delim)
		if end < 0 {
			return "", fmt.Errorf("line %d: heredoc %s is not terminated", i+1, delim)
		}
		var body strings.Builder
		for _, line := range lines[i+1 : i+1+end] {
			body.WriteString(line + "\n")
		}
		bodies = append(bodies, body.String())
		i += end + 1
	}
	switch len(bodies) {
	case 0:
		return "", fmt.Errorf("no COPY heredoc writes %s", dest)
	case 1:
		return bodies[0], nil
	default:
		return "", fmt.Errorf("%d COPY heredocs write %s, want 1", len(bodies), dest)
	}
}
