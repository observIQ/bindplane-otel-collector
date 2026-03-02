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

package worker

import "io"

// countingReader is a reader that counts the number of bytes read.
type countingReader struct {
	reader io.Reader
	offset int64
}

// Offset returns the number of bytes read.
func (r *countingReader) Offset() int64 {
	return r.offset
}

// Read reads the given number of bytes and updates the offset.
func (r *countingReader) Read(p []byte) (n int, err error) {
	n, err = r.reader.Read(p)
	r.offset += int64(n)
	return n, err
}
