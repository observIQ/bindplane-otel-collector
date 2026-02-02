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

// Package client provides a client for the pebble database
package client

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/cockroachdb/pebble"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/extension/xextension/storage"
	"go.uber.org/zap"
)

// Client is the interface for the pebble client
type Client interface {
	storage.Client
	// Compact performs a compaction operation on the database to reclaim space from deleted entries.
	Compact(parallel bool) error
}

// DBClient is the interface for the pebble database
type DBClient interface {
	Get(key []byte) ([]byte, io.Closer, error)
	Set(key []byte, value []byte, opts *pebble.WriteOptions) error
	Delete(key []byte, opts *pebble.WriteOptions) error
	NewBatch() *pebble.Batch
	// Flush for the memtable is written to disk for Pebble
	Flush() error
	Close() error
	Compact(start, end []byte, parallel bool) error
}

type client struct {
	db DBClient

	logger       *zap.Logger
	path         string
	doneChan     chan struct{}
	writeOptions *pebble.WriteOptions
	readOnly     bool
}

// Options are the options for opening the pebble database
type Options struct {
	Sync                  bool
	CacheSize             int64
	CompactionConcurrency int
}

// ErrClientClosing is supposed to be returned when the client is closing and not allowing new operations to be performed
var ErrClientClosing = errors.New("client is closing")

// NewClient creates a new client for the pebble database
func NewClient(path string, logger *zap.Logger, options *Options) (Client, error) {
	c := &client{
		path:   path,
		logger: logger,
	}

	writeOptions := &pebble.WriteOptions{}
	if options.Sync {
		writeOptions.Sync = true
	}
	c.writeOptions = writeOptions

	openOptions := &pebble.Options{}
	if options.CacheSize > 0 {
		openOptions.Cache = pebble.NewCache(options.CacheSize)
	}
	openOptions.Logger = logger.Sugar()

	db, err := pebble.Open(path, openOptions)
	if err != nil {
		return nil, fmt.Errorf("unable to open database at %s: %w", path, err)
	}
	c.db = db
	return c, nil
}

// Get gets a key from the pebble database
func (c *client) Get(_ context.Context, key string) ([]byte, error) {
	if c.isDone() {
		return nil, ErrClientClosing
	}

	val, closer, err := c.db.Get([]byte(key))
	if err != nil {
		if errors.Is(err, pebble.ErrNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("error getting key %s: %w", key, err)
	}
	defer closer.Close()

	if val == nil {
		return nil, nil
	}

	return val, nil
}

// isDone checks if the client has its done channel closed
func (c *client) isDone() bool {
	select {
	case <-c.doneChan:
		return true
	default:
		return false
	}
}

// Set sets a key in the pebble database
func (c *client) Set(_ context.Context, key string, value []byte) error {
	if c.isDone() {
		return ErrClientClosing
	}
	err := c.db.Set([]byte(key), value, c.writeOptions)
	if err != nil {
		return fmt.Errorf("error setting key %s: %w", key, err)
	}
	return nil
}

// Delete deletes a key from the pebble database
func (c *client) Delete(_ context.Context, key string) error {
	if c.isDone() {
		return ErrClientClosing
	}

	err := c.db.Delete([]byte(key), c.writeOptions)
	if err != nil {
		return fmt.Errorf("error deleting key %s: %w", key, err)
	}
	return nil
}

func (c *client) Batch(ctx context.Context, ops ...*storage.Operation) error {
	var wb *pebble.Batch
	var err error
	for _, op := range ops {
		if c.isDone() {
			err = ErrClientClosing
			break
		}

		var writes bool
		switch op.Type {
		case storage.Set, storage.Delete:
			writes = true
			if c.readOnly {
				return errors.New("database is in read-only mode")
			}
		}

		if writes && wb == nil {
			wb = c.db.NewBatch()
			defer wb.Close()
		}

		var err error
		switch op.Type {
		case storage.Set:
			err = wb.Set([]byte(op.Key), op.Value, c.writeOptions)
		case storage.Delete:
			err = wb.Delete([]byte(op.Key), c.writeOptions)
		case storage.Get:
			value, err := c.Get(ctx, op.Key)
			if err != nil {
				return fmt.Errorf("error getting key %s: %w", op.Key, err)
			}
			op.Value = value
		default:
			return errors.New("wrong operation type")
		}

		if err != nil {
			return fmt.Errorf("failed to perform %s on %s item: %w", typeString(op.Type), op.Key, err)
		}
	}
	if wb != nil {
		return wb.Commit(c.writeOptions)
	}

	if err != nil {
		return err
	}
	return nil
}

func typeString(t storage.OpType) string {
	switch t {
	case storage.Set:
		return "set"
	case storage.Delete:
		return "delete"
	case storage.Get:
		return "get"
	}
	return "unknown"
}

func (c *client) Start(_ context.Context, _ component.Host) error {
	c.doneChan = make(chan struct{})
	return nil
}

func (c *client) Close(_ context.Context) error {
	if c.doneChan != nil {
		close(c.doneChan)
	}

	err := c.db.Flush()
	if err != nil {
		return fmt.Errorf("flush db: %w", err)
	}

	return c.db.Close()
}

// Compact performs a compaction operation on the database to reclaim space from deleted entries.
// Note: Compaction is I/O intensive and may impact performance during operation so we should only sparsely run it if necessary.
// Note: in v2 of Pebble we will use the context
func (c *client) Compact(parallel bool) error {
	c.logger.Debug("compacting database to reclaim space", zap.String("path", c.path))
	// just compacting the entire database for now, there may be a better way of doing this but this is a starting point.
	// tried looking into how cockroachdb does it but they have different lifeclycle
	return c.db.Compact([]byte{}, []byte{0xff, 0xff, 0xff, 0xff}, parallel)
}
