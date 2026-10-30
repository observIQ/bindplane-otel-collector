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

package file

import (
	"io/fs"
	"os"
	"syscall"

	"go.uber.org/zap"
)

// lchown is swapped out in tests.
var lchown = os.Lchown

// keepOwner gives path the owner and group of from. An updater running as the
// unprivileged collector user can't chown to root, so a failure is logged
// instead of failing the copy.
func keepOwner(logger *zap.Logger, path string, from fs.FileInfo) {
	st, ok := from.Sys().(*syscall.Stat_t)
	if !ok {
		return
	}
	if err := lchown(path, int(st.Uid), int(st.Gid)); err != nil {
		logger.Warn("Failed to keep file owner", zap.String("path", path), zap.Error(err))
	}
}
