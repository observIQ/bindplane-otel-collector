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
