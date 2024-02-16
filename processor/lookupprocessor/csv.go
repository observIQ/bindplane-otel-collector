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

package lookupprocessor

import (
	"encoding/csv"
	"errors"
	"fmt"
	"os"
	"sync"
)

var (
	// errCSVNotLoaded is the error for when the csv is not loaded
	errCSVNotLoaded = errors.New("csv not loaded")
	// errKeyNotFound is the error for when the key is not found
	errKeyNotFound = errors.New("key not found")
	// errNoRecords is the error for when there are no records to parse
	errNoRecords = errors.New("no records to parse")
	// errLookupColumnNotFound is the error for when the lookup column is not found
	errLookupColumnNotFound = errors.New("lookup column not found")
)

// CSVFile is a file that contains csv data
type CSVFile struct {
	filepath     string
	lookupColumn string
	data         map[string]map[string]string
	mux          *sync.RWMutex
}

// Load loads the csv into memory
func (c *CSVFile) Load() error {
	c.mux.Lock()
	defer c.mux.Unlock()

	file, err := os.Open(c.filepath)
	if err != nil {
		return fmt.Errorf("open file: %w", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return fmt.Errorf("read all: %w", err)
	}

	data, err := indexRecords(records, c.lookupColumn)
	if err != nil {
		return fmt.Errorf("index records: %w", err)
	}

	c.data = data
	return nil
}

// Lookup returns a row of data that matches the key in the provided column
func (c *CSVFile) Lookup(key string) (map[string]string, error) {
	c.mux.RLock()
	defer c.mux.RUnlock()

	if c.data == nil {
		return nil, errCSVNotLoaded
	}

	results, ok := c.data[key]
	if !ok {
		return nil, errKeyNotFound
	}

	return results, nil
}

// indexRecords indexes the records by the lookup column
func indexRecords(records [][]string, lookupColumn string) (map[string]map[string]string, error) {
	if len(records) == 0 {
		return nil, errNoRecords
	}

	headers := records[0]
	lookupIndex, err := findLookupIndex(headers, lookupColumn)
	if err != nil {
		return nil, fmt.Errorf("find lookup index: %w", err)
	}

	result := make(map[string]map[string]string)
	for _, record := range records[1:] {
		lookupKey := record[lookupIndex]
		result[lookupKey] = make(map[string]string)
		for i, value := range record {
			// Skip the lookup column
			if i == lookupIndex {
				continue
			}

			result[lookupKey][headers[i]] = value
		}
	}

	return result, nil
}

// findLookupIndex finds the index of the lookup column
func findLookupIndex(headers []string, lookupColumn string) (int, error) {
	for i, header := range headers {
		if header == lookupColumn {
			return i, nil
		}
	}

	return -1, errLookupColumnNotFound
}

// NewCSVFile creates a new CSVFile
func NewCSVFile(filepath string, lookupColumn string) *CSVFile {
	return &CSVFile{
		mux:          &sync.RWMutex{},
		filepath:     filepath,
		lookupColumn: lookupColumn,
	}
}
