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
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// releaseBaseURL is where released packages are published, see
// scripts/install/install_unix.sh.
const releaseBaseURL = "https://bdot.bindplane.com"

// downloadMu serializes downloads so concurrent scenarios that need the same
// release do not download it twice.
var downloadMu sync.Mutex

// releasePackage returns the host path of a released package, downloading
// it into the user cache directory on first use.
func releasePackage(t *testing.T, version, arch string, format pkgFormat) string {
	t.Helper()

	name := fmt.Sprintf("%s_v%s_linux_%s.%s", packageName, version, arch, format)
	cacheDir, err := os.UserCacheDir()
	require.NoError(t, err)
	dst := filepath.Join(cacheDir, "bdot-pkgtest", "v"+version, name)

	downloadMu.Lock()
	defer downloadMu.Unlock()

	if _, err := os.Stat(dst); err == nil {
		t.Logf("using cached release package %s", dst)
		return dst
	}

	url := fmt.Sprintf("%s/v%s/%s", releaseBaseURL, version, name)
	t.Logf("downloading %s", url)
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Minute)
	defer cancel()
	require.NoError(t, download(ctx, url, dst), "download release %s", version)
	return dst
}

// download fetches url into dst. It writes to a temporary file first so
// that an interrupted download is never mistaken for a cached package.
func download(ctx context.Context, url, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o750); err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("GET %s: %s", url, resp.Status)
	}

	tmp := dst + ".partial"
	f, err := os.Create(tmp) // #nosec G304 -- path is under the user cache dir
	if err != nil {
		return err
	}
	if _, err := io.Copy(f, resp.Body); err != nil {
		_ = f.Close()
		_ = os.Remove(tmp)
		return fmt.Errorf("GET %s: %w", url, err)
	}
	if err := f.Close(); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return os.Rename(tmp, dst)
}
