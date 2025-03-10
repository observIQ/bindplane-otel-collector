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

package observiq

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"crypto/sha256"
	"crypto/subtle"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/observiq/bindplane-otel-collector/opamp"
	"github.com/open-telemetry/opamp-go/protobufs"
	"go.uber.org/zap"
)

const extractFolder = "latest"
const maxArchiveObjectByteSize = 1000000000

// Ensure interface is satisfied
var _ opamp.DownloadableFileManager = (*DownloadableFileManager)(nil)

// DownloadableFileManager handles DownloadableFile's from a PackagesAvailable message
type DownloadableFileManager struct {
	tmpPath string
	logger  *zap.Logger
}

// newDownloadableFileManager creates a new OpAmp DownloadableFileManager
func newDownloadableFileManager(logger *zap.Logger, tmpPath string) *DownloadableFileManager {
	return &DownloadableFileManager{
		tmpPath: filepath.Clean(tmpPath),
		logger:  logger,
	}
}

// FetchAndExtractArchive fetches the archive at the specified URL, placing it into dir.
// It then checks to see if it matches the "expectedHash", a hex-encoded string representing the expected sha256 sum of the file.
// If it matches, the archive is extracted into the $dir/latest directory.
// If the archive cannot be extracted, downloaded, or verified, then an error is returned.
func (m DownloadableFileManager) FetchAndExtractArchive(file *protobufs.DownloadableFile) error {
	archiveFilePath, err := getOutputFilePath(m.tmpPath, file.GetDownloadUrl())
	if err != nil {
		return fmt.Errorf("failed to determine archive download path: %w", err)
	}

	if err := m.downloadFile(file.GetDownloadUrl(), archiveFilePath); err != nil {
		return fmt.Errorf("failed to download file: %w", err)
	}

	extractPath := filepath.Join(m.tmpPath, extractFolder)

	if err := m.verifyContentHash(archiveFilePath, file.GetContentHash()); err != nil {
		return fmt.Errorf("content hash could not be verified: %w", err)
	}

	// Clean the "latest" dir before extraction
	if err := os.RemoveAll(extractPath); err != nil {
		return fmt.Errorf("error cleaning archive extraction target path: %w", err)
	}

	if err := unarchive(archiveFilePath, extractPath); err != nil {
		return fmt.Errorf("failed to extract file: %w", err)
	}

	return nil
}

// Downloads the file into the outPath, truncating the file if it already exists
func (m DownloadableFileManager) downloadFile(downloadURL string, outPath string) error {
	//#nosec G107 HTTP request must be dynamic based on input
	resp, err := http.Get(downloadURL)
	if err != nil {
		return fmt.Errorf("could not GET url: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			m.logger.Warn("Failed to close response body while downloading file", zap.String("URL", downloadURL), zap.Error(err))
		}
	}()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("got non-200 status code (%d)", resp.StatusCode)
	}

	outPathClean := filepath.Clean(outPath)
	f, err := os.OpenFile(outPathClean, os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0600)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer func() {
		err := f.Close()
		if err != nil {
			m.logger.Warn("Failed to close file", zap.Error(err))
		}
	}()

	if _, err = io.Copy(f, resp.Body); err != nil {
		return fmt.Errorf("failed to copy request body to file: %w", err)
	}

	return nil
}

// getOutputFilePath gets the output path relative to the base dir for the archive from the given URL.
func getOutputFilePath(basePath, downloadURL string) (string, error) {
	err := os.MkdirAll(basePath, 0700)
	if err != nil {
		return "", fmt.Errorf("problem with base url: %w", err)
	}

	url, err := url.Parse(downloadURL)
	if err != nil {
		return "", fmt.Errorf("cannot parse url: %w", err)
	}

	if url.Path == "" {
		return "", errors.New("input url must have path")
	}

	return filepath.Join(basePath, filepath.Base(url.Path)), nil
}

func (m DownloadableFileManager) verifyContentHash(contentPath string, expectedFileHash []byte) error {
	// Hash file at contentPath using sha256
	fileHash := sha256.New()
	contentPathClean := filepath.Clean(contentPath)

	f, err := os.Open(contentPathClean)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer func() {
		err := f.Close()
		if err != nil {
			m.logger.Warn("Failed to close file", zap.Error(err))
		}
	}()

	if _, err = io.Copy(fileHash, f); err != nil {
		return fmt.Errorf("failed to calculate file hash: %w", err)
	}

	actualContentHash := fileHash.Sum(nil)
	if subtle.ConstantTimeCompare(expectedFileHash, actualContentHash) == 0 {
		return errors.New("file hash did not match expected")
	}

	return nil
}

// CleanupArtifacts removes previous installation artifacts by removing the temporary directory.
func (m DownloadableFileManager) CleanupArtifacts() {
	if err := os.RemoveAll(m.tmpPath); err != nil {
		m.logger.Error("Failed to remove temporary directory", zap.Error(err))
	}
}

// unarchive will unpack the package at archivePath(.tar.gz or .zip) into the directory found at extractPath
func unarchive(archivePath, extractPath string) error {
	if strings.HasSuffix(archivePath, ".tar.gz") {
		// Handle tar.gz files
		if err := extractTarGz(archivePath, extractPath); err != nil {
			return fmt.Errorf("extract tar.gz: %w", err)
		}
	} else if strings.HasSuffix(archivePath, ".zip") {
		// Handle zip files
		if err := extractZip(archivePath, extractPath); err != nil {
			return fmt.Errorf("extract .zip: %w", err)
		}
	} else {
		return fmt.Errorf("unsupported file type: %s", archivePath)
	}

	return nil
}

// extractTarGz will extract the .tar package at archivePath into the dir at extractPath
func extractTarGz(archivePath, extractPath string) error {
	if err := os.MkdirAll(extractPath, 0750); err != nil {
		return fmt.Errorf("mkdir extract path: %w", err)
	}

	archivePathClean := filepath.Clean(archivePath)
	file, err := os.Open(archivePathClean)
	if err != nil {
		return fmt.Errorf("open archive package: %w", err)
	}
	defer file.Close()

	gzipReader, err := gzip.NewReader(file)
	if err != nil {
		return fmt.Errorf("new gzip reader: %w", err)
	}
	defer gzipReader.Close()

	tarReader := tar.NewReader(gzipReader)

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("read next tarball header: %w", err)
		}

		outputPath, err := sanitizeArchivePath(extractPath, header.Name)
		if err != nil {
			return fmt.Errorf("sanitize archive path: %w", err)
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(outputPath, os.FileMode(header.Mode)); err != nil {
				return fmt.Errorf("mkdir: %w", err)
			}

		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(outputPath), 0700); err != nil {
				return fmt.Errorf("create file's dir: %w", err)
			}

			outputPathClean := filepath.Clean(outputPath)
			outFile, err := os.OpenFile(outputPathClean, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, os.FileMode(header.Mode))
			if err != nil {
				return fmt.Errorf("open output file: %w", err)
			}
			defer outFile.Close()

			_, err = io.CopyN(outFile, tarReader, maxArchiveObjectByteSize)
			if err != nil && err != io.EOF {
				return fmt.Errorf("write to file: %w", err)
			}

		default:
			fmt.Printf("Unsupported type: %v in %s\n", header.Typeflag, header.Name)
		}
	}
	return nil
}

// extractZip will extract the .zip package at archivePath into the dir at extractPath
func extractZip(archivePath, extractPath string) error {
	// Ensure the output directory exists
	if err := os.MkdirAll(extractPath, 0750); err != nil {
		return fmt.Errorf("mkdir extract path: %w", err)
	}

	r, err := zip.OpenReader(archivePath)
	if err != nil {
		return fmt.Errorf("new zip reader: %w", err)
	}
	defer r.Close()

	for _, f := range r.File {
		outputPath, err := sanitizeArchivePath(extractPath, f.Name)
		if err != nil {
			return fmt.Errorf("sanitize archive path: %w", err)
		}

		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(outputPath, f.Mode()); err != nil {
				return fmt.Errorf("mkdir: %w", err)
			}
			continue
		}

		if err := os.MkdirAll(filepath.Dir(outputPath), 0700); err != nil {
			return fmt.Errorf("create file's dir: %w", err)
		}

		outputPathClean := filepath.Clean(outputPath)
		outFile, err := os.OpenFile(outputPathClean, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return fmt.Errorf("open output file: %w", err)
		}
		defer outFile.Close()

		rc, err := f.Open()
		if err != nil {
			return fmt.Errorf("open source file: %w", err)
		}
		defer rc.Close()

		_, err = io.CopyN(outFile, rc, maxArchiveObjectByteSize)
		if err != nil && err != io.EOF {
			return fmt.Errorf("write source file to output file: %w", err)
		}
	}
	return nil
}

func sanitizeArchivePath(dir, file string) (string, error) {
	s := filepath.Join(dir, file)
	if strings.HasPrefix(s, filepath.Clean(dir)) {
		return s, nil
	}
	return "", fmt.Errorf("content filepath is tainted: %q", file)
}
