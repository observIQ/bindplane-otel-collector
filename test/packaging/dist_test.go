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

//go:build packaging

package packaging

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// distMetadata is the subset of goreleaser's dist/metadata.json used by the
// suite.
type distMetadata struct {
	ProjectName string `json:"project_name"`
	Tag         string `json:"tag"`
	PreviousTag string `json:"previous_tag"`
	Version     string `json:"version"`
	Commit      string `json:"commit"`
	Date        string `json:"date"`
}

// distArtifact is the subset of a goreleaser dist/artifacts.json entry used
// by the suite.
type distArtifact struct {
	Name   string `json:"name"`
	Path   string `json:"path"`
	Goos   string `json:"goos"`
	Goarch string `json:"goarch"`
	Type   string `json:"type"`
	Extra  struct {
		Ext      string `json:"Ext"`
		Checksum string `json:"Checksum"`
	} `json:"extra"`
}

// dist is a goreleaser output directory.
type dist struct {
	dir       string
	metadata  distMetadata
	artifacts []distArtifact
}

func loadDist(dir string) (*dist, error) {
	d := &dist{dir: dir}
	if err := readJSON(filepath.Join(dir, "metadata.json"), &d.metadata); err != nil {
		return nil, err
	}
	if err := readJSON(filepath.Join(dir, "artifacts.json"), &d.artifacts); err != nil {
		return nil, err
	}
	return d, nil
}

func readJSON(path string, v any) error {
	b, err := os.ReadFile(path) // #nosec G304 -- path is the configured dist dir
	if err != nil {
		return fmt.Errorf("read %s (run `make release-test` first): %w", path, err)
	}
	if err := json.Unmarshal(b, v); err != nil {
		return fmt.Errorf("parse %s: %w", path, err)
	}
	return nil
}

// linuxPackage returns the host path of the Linux package for the given
// arch and format, after verifying its checksum.
func (d *dist) linuxPackage(arch string, format pkgFormat) (string, error) {
	for _, a := range d.artifacts {
		if a.Type != "Linux Package" || a.Goos != "linux" || a.Goarch != arch || a.Extra.Ext != "."+string(format) {
			continue
		}
		path := filepath.Join(d.dir, a.Name)
		if err := verifyChecksum(path, a.Extra.Checksum); err != nil {
			return "", err
		}
		return path, nil
	}
	return "", fmt.Errorf("no linux/%s %s package in %s/artifacts.json; run `make release-test` to build packages for this architecture", arch, format, d.dir)
}

// verifyChecksum checks a file against a goreleaser checksum in the
// "sha256:<hex>" form.
func verifyChecksum(path, checksum string) error {
	algo, want, ok := strings.Cut(checksum, ":")
	if !ok || algo != "sha256" {
		return fmt.Errorf("unsupported checksum %q for %s", checksum, path)
	}
	f, err := os.Open(path) // #nosec G304 -- path comes from the configured dist dir
	if err != nil {
		return fmt.Errorf("open package: %w", err)
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return fmt.Errorf("hash %s: %w", path, err)
	}
	if got := hex.EncodeToString(h.Sum(nil)); got != want {
		return fmt.Errorf("checksum mismatch for %s: got sha256:%s, artifacts.json has %s; dist is stale or corrupt, re-run `make release-test`", path, got, checksum)
	}
	return nil
}
