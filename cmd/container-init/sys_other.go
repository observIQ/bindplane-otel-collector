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

//go:build !unix

package main

import (
	"errors"
	"io/fs"
)

// unsupportedFS implements fsOps on platforms without unix ownership. The
// container images are linux only, so this exists to keep the package
// buildable elsewhere.
type unsupportedFS struct{}

func newFS() fsOps { return unsupportedFS{} }

func (unsupportedFS) Lchown(string, uint32, uint32) error {
	return errors.New("-chown is only supported on unix")
}

func (unsupportedFS) Owner(string, fs.FileInfo) (uint32, uint32, bool) { return 0, 0, false }

func (unsupportedFS) Device(string, fs.FileInfo) (uint64, bool) { return 0, false }

func describeProcess() string { return "" }
