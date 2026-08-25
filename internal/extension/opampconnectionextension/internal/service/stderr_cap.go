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

package service

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

// The stderr file (observiq_collector.err) captures output that does not go
// through the zap logger (runtime panics, high-volume component error spew).
// It historically had no size cap and has filled OS drives (10+ GB observed).
// These constants bound it to cap + 1 backup.
// ponytail: hardcoded cap/backups; expose via config if a customer ever needs to tune it.
const (
	stderrMaxBytes      = 10 * 1024 * 1024
	stderrCheckInterval = 10 * time.Second
)

// rollStderrFile rolls an existing, non-empty stderr file to "<path>.1"
// (replacing any previous backup) so it can't accumulate across service
// restarts. It must be called before the file is opened as the stderr target:
// on Windows, renaming a file our own stderr handle holds open fails with a
// sharing violation (Go opens files without FILE_SHARE_DELETE).
func rollStderrFile(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("stat stderr file: %w", err)
	}
	if info.Size() == 0 {
		return nil
	}
	if err := os.Rename(path, path+".1"); err != nil {
		return fmt.Errorf("roll stderr file: %w", err)
	}
	return nil
}

// capStderrFile bounds the stderr file while it is held open as the OS stderr
// handle: if it has grown to at least threshold bytes, its contents are
// copied aside to "<path>.1" and the file is truncated in place. Truncating
// (rather than renaming) keeps the already-bound stderr handle valid: the
// handle appends at EOF (O_APPEND / FILE_APPEND_DATA), so writes continue
// correctly after truncation. Writes landing between the copy and the
// truncate are lost — the goal is "bounded", not "exact".
func capStderrFile(path string, threshold int64) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("stat stderr file: %w", err)
	}
	if info.Size() < threshold {
		return nil
	}

	src, err := os.Open(filepath.Clean(path))
	if err != nil {
		return fmt.Errorf("open stderr file: %w", err)
	}
	defer src.Close()

	dst, err := os.OpenFile(filepath.Clean(path+".1"), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("open stderr backup file: %w", err)
	}
	if _, err := io.Copy(dst, src); err != nil {
		_ = dst.Close()
		return fmt.Errorf("copy stderr file to backup: %w", err)
	}
	if err := dst.Close(); err != nil {
		return fmt.Errorf("close stderr backup file: %w", err)
	}

	if err := os.Truncate(path, 0); err != nil {
		return fmt.Errorf("truncate stderr file: %w", err)
	}
	return nil
}

// watchStderrFile periodically caps the stderr file for the life of the
// process. Errors are best-effort: there is nowhere safe to report them
// repeatedly (stderr is the file being managed), so ticks that fail are
// simply retried on the next interval.
func watchStderrFile(path string) {
	ticker := time.NewTicker(stderrCheckInterval)
	defer ticker.Stop()
	for range ticker.C {
		_ = capStderrFile(path, stderrMaxBytes)
	}
}
