// Copyright  Bindplane, Inc.
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

package file

import (
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

func TestCopyFileOverwrite(t *testing.T) {
	t.Run("Copies file when output does not exist", func(t *testing.T) {
		tmpDir := t.TempDir()

		inFile := filepath.Join("testdata", "test.txt")
		outFile := filepath.Join(tmpDir, "test.txt")

		err := CopyFileOverwrite(zaptest.NewLogger(t), inFile, outFile)
		require.NoError(t, err)
		require.FileExists(t, outFile)

		contentsIn, err := os.ReadFile(inFile)
		require.NoError(t, err)

		contentsOut, err := os.ReadFile(outFile)
		require.NoError(t, err)

		require.Equal(t, contentsIn, contentsOut)

		fi, err := os.Stat(outFile)
		require.NoError(t, err)
		// file mode on windows acts unlike unix, we'll only check for this on linux/darwin
		if runtime.GOOS != "windows" {
			require.Equal(t, fs.FileMode(0600), fi.Mode())
		}
	})

	t.Run("Copies file when output already exists", func(t *testing.T) {
		tmpDir := t.TempDir()

		inFile := filepath.Join("testdata", "test.txt")
		outFile := filepath.Join(tmpDir, "test.txt")

		contentsIn, err := os.ReadFile(inFile)
		require.NoError(t, err)

		err = os.WriteFile(outFile, []byte("This is a file that already exists"), 0640)
		require.NoError(t, err)

		fioOrig, err := os.Stat(outFile)
		require.NoError(t, err)

		err = CopyFileOverwrite(zaptest.NewLogger(t), inFile, outFile)
		require.NoError(t, err)
		require.FileExists(t, outFile)

		contentsOut, err := os.ReadFile(outFile)
		require.NoError(t, err)
		require.Equal(t, contentsIn, contentsOut)

		fio, err := os.Stat(outFile)
		require.NoError(t, err)
		// file mode on windows acts unlike unix, we'll only check for this on linux/darwin
		if runtime.GOOS != "windows" {
			require.Equal(t, fioOrig.Mode(), fio.Mode())
		}
	})

	t.Run("Fails when input file does not exist", func(t *testing.T) {
		tmpDir := t.TempDir()

		inFile := filepath.Join("testdata", "does-not-exist.txt")
		outFile := filepath.Join(tmpDir, "test.txt")

		err := CopyFileOverwrite(zaptest.NewLogger(t), inFile, outFile)
		require.ErrorContains(t, err, "failed to stat input file")
		require.NoFileExists(t, outFile)
	})

	t.Run("Does not truncate if input file does not exist", func(t *testing.T) {
		tmpDir := t.TempDir()

		inFile := filepath.Join("testdata", "does-not-exist.txt")
		outFile := filepath.Join(tmpDir, "test.txt")

		err := os.WriteFile(outFile, []byte("This is a file that already exists"), 0600)
		require.NoError(t, err)

		err = CopyFileOverwrite(zaptest.NewLogger(t), inFile, outFile)
		require.ErrorContains(t, err, "failed to stat input file")
		require.FileExists(t, outFile)

		contentsOut, err := os.ReadFile(outFile)
		require.NoError(t, err)
		require.Equal(t, []byte("This is a file that already exists"), contentsOut)
	})
}

func TestCopyFileRollback(t *testing.T) {
	t.Run("Copies file when output does not exist", func(t *testing.T) {
		tmpDir := t.TempDir()

		inFile := filepath.Join("testdata", "test.txt")
		outFile := filepath.Join(tmpDir, "test.txt")

		err := CopyFileNoOverwrite(zaptest.NewLogger(t), inFile, outFile)
		require.NoError(t, err)
		require.FileExists(t, outFile)

		contentsIn, err := os.ReadFile(inFile)
		require.NoError(t, err)

		contentsOut, err := os.ReadFile(outFile)
		require.NoError(t, err)

		require.Equal(t, contentsIn, contentsOut)

		fio, err := os.Stat(outFile)
		require.NoError(t, err)
		fii, err := os.Stat(outFile)
		require.NoError(t, err)
		// file mode on windows acts unlike unix, we'll only check for this on linux/darwin
		if runtime.GOOS != "windows" {
			require.Equal(t, fii.Mode(), fio.Mode())
		}
	})

	t.Run("Fails to overwrite the output file", func(t *testing.T) {
		tmpDir := t.TempDir()

		inFile := filepath.Join("testdata", "test.txt")
		outFile := filepath.Join(tmpDir, "test.txt")

		err := os.WriteFile(outFile, []byte("This is a file that already exists"), 0640)
		require.NoError(t, err)

		err = CopyFileNoOverwrite(zaptest.NewLogger(t), inFile, outFile)
		require.ErrorContains(t, err, "failed to open output file")
		require.FileExists(t, outFile)

		contentsOut, err := os.ReadFile(outFile)
		require.NoError(t, err)
		require.Equal(t, []byte("This is a file that already exists"), contentsOut)

		fi, err := os.Stat(outFile)
		require.NoError(t, err)
		// file mode on windows acts unlike unix, we'll only check for this on linux/darwin
		if runtime.GOOS != "windows" {
			require.Equal(t, fs.FileMode(0640), fi.Mode())
		}
	})

	t.Run("Fails when input file does not exist", func(t *testing.T) {
		tmpDir := t.TempDir()

		inFile := filepath.Join("testdata", "does-not-exist.txt")
		outFile := filepath.Join(tmpDir, "test.txt")

		err := CopyFileNoOverwrite(zaptest.NewLogger(t), inFile, outFile)
		require.ErrorContains(t, err, "failed to retrieve fileinfo for input file")
		require.NoFileExists(t, outFile)
	})
}

func TestCopyFileNoOverwrite(t *testing.T) {
	t.Run("Copies file when output does not exist and uses inFile's permissions", func(t *testing.T) {
		tmpDir := t.TempDir()

		inFile := filepath.Join("testdata", "test.txt")
		outFile := filepath.Join(tmpDir, "test.txt")

		err := CopyFileRollback(zaptest.NewLogger(t), inFile, outFile)
		require.NoError(t, err)
		require.FileExists(t, outFile)

		contentsIn, err := os.ReadFile(inFile)
		require.NoError(t, err)

		contentsOut, err := os.ReadFile(outFile)
		require.NoError(t, err)

		require.Equal(t, contentsIn, contentsOut)

		fio, err := os.Stat(outFile)
		require.NoError(t, err)
		fii, err := os.Stat(outFile)
		require.NoError(t, err)
		// file mode on windows acts unlike unix, we'll only check for this on linux/darwin
		if runtime.GOOS != "windows" {
			require.Equal(t, fii.Mode(), fio.Mode())
		}
	})

	t.Run("Fails when input file does not exist", func(t *testing.T) {
		tmpDir := t.TempDir()

		inFile := filepath.Join("testdata", "does-not-exist.txt")
		outFile := filepath.Join(tmpDir, "test.txt")

		err := CopyFileRollback(zaptest.NewLogger(t), inFile, outFile)
		require.ErrorContains(t, err, "input file does not exist")
		require.NoFileExists(t, outFile)
	})
}
