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

// Package client contains the implementation of the badger storage client
package client

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/dgraph-io/badger/v4"
	"go.opentelemetry.io/collector/extension/xextension/storage"
	"go.uber.org/zap"
)

// Client is the interface for the badger storage client
type Client interface {
	storage.Client
	RunValueLogGC(discardRatio float64) error
	GetDiskUsage() (DiskUsage, error)
	GetOperationCounts() OperationCounts
}

// Options is the interface for the badger storage client options
type Options struct {
	// whether or not to use fsync for badger
	SyncWrites bool

	// the size of the memory table to use for badger
	MemTableSize int64

	// the size of the block cache to use for badger
	BlockCacheSize int64

	// the maximum size of a value log file in bytes
	ValueLogFileSize int64

	// number of background compaction workers (higher = more aggressive compaction)
	NumCompactors int

	// number of L0 tables that triggers compaction (lower = earlier compaction)
	NumLevelZeroTables int

	// number of L0 tables that stalls writes until compaction catches up
	NumLevelZeroTablesStall int
}

var _ Client = (*client)(nil)

// OperationCounts holds the count of operations performed by the client
type OperationCounts struct {
	Get    int64
	Set    int64
	Delete int64
}

type client struct {
	db *badger.DB

	// operation counters (use atomic operations)
	opGet    atomic.Int64
	opSet    atomic.Int64
	opDelete atomic.Int64

	logger *zap.Logger
}

// NewClient creates a new Badger client for use in the extension
func NewClient(path string, opts *Options, logger *zap.Logger) (Client, error) {
	c := &client{
		logger: logger,
	}

	options := badger.DefaultOptions(path)
	if opts.SyncWrites {
		options = options.WithSyncWrites(true)
	}

	if opts.MemTableSize > 0 {
		options = options.WithMemTableSize(opts.MemTableSize)
	}

	if opts.BlockCacheSize > 0 {
		options = options.WithBlockCacheSize(int64(opts.BlockCacheSize))
	}

	if opts.ValueLogFileSize > 0 {
		options = options.WithValueLogFileSize(opts.ValueLogFileSize)
	}

	if opts.NumCompactors > 0 {
		options = options.WithNumCompactors(opts.NumCompactors)
	}

	if opts.NumLevelZeroTables > 0 {
		options = options.WithNumLevelZeroTables(opts.NumLevelZeroTables)
	}

	if opts.NumLevelZeroTablesStall > 0 {
		options = options.WithNumLevelZeroTablesStall(opts.NumLevelZeroTablesStall)
	}

	// override the logger to hide logs from badger if they try to log anything
	options = options.WithLogger(&badgerNopLogger{logger: zap.NewNop().Sugar()})
	db, err := badger.Open(options)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}
	c.db = db

	return c, nil
}

// Get will retrieve data from storage that corresponds to the specified key
func (c *client) Get(_ context.Context, key string) ([]byte, error) {
	c.opGet.Add(1)

	tx := c.db.NewTransaction(false)
	defer tx.Discard()

	item, err := tx.Get([]byte(key))
	if errors.Is(err, badger.ErrKeyNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get item: %w", err)
	}

	val, err := item.ValueCopy(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to copy value: %w", err)
	}
	if len(val) == 0 {
		return nil, nil
	}
	return val, nil
}

// Set will store data. The data can be retrieved using the same key
func (c *client) Set(_ context.Context, key string, value []byte) error {
	c.opSet.Add(1)

	tx := c.db.NewTransaction(true)
	defer tx.Discard()

	err := tx.Set([]byte(key), value)
	if err != nil {
		return fmt.Errorf("failed to set item: %w", err)
	}
	return tx.Commit()
}

func (c *client) Delete(_ context.Context, key string) error {
	c.opDelete.Add(1)

	tx := c.db.NewTransaction(true)
	defer tx.Discard()

	err := tx.Delete([]byte(key))
	if err != nil {
		return fmt.Errorf("failed to delete item: %w", err)
	}
	return tx.Commit()
}

func (c *client) Batch(ctx context.Context, ops ...*storage.Operation) error {
	var wb *badger.WriteBatch
	for _, op := range ops {
		writes := false
		if op.Type == storage.Set || op.Type == storage.Delete {
			writes = true
		}

		if writes && wb == nil {
			wb = c.db.NewWriteBatch()
			defer wb.Cancel()
		}

		var err error
		var value []byte
		switch op.Type {
		case storage.Set:
			c.opSet.Add(1)
			err = wb.Set([]byte(op.Key), op.Value)
		case storage.Delete:
			c.opDelete.Add(1)
			err = wb.Delete([]byte(op.Key))
		case storage.Get:
			c.opGet.Add(1)
			value, err = c.Get(ctx, op.Key)
			if err == nil {
				op.Value = value
			}
		default:
			return errors.New("wrong operation type")
		}
		if err != nil {
			return fmt.Errorf("failed to perform %s on %s item: %w", typeString(op.Type), op.Key, err)
		}
	}

	if wb != nil {
		if err := wb.Flush(); err != nil {
			return fmt.Errorf("failed to flush write batch: %w", err)
		}
	}

	return nil
}

func (c *client) Close(ctx context.Context) error {
	err := c.db.Close()
	if err != nil {
		return fmt.Errorf("failed to close badger client: %w", err)
	}

	isFullyClosed := make(chan struct{})
	go func() {
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()
		defer close(isFullyClosed)
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if c.db.IsClosed() {
					return
				}
			}
		}
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-isFullyClosed:
		return nil
	}
}

// RunValueLogGC runs the value log garbage collection
// its in an infinite loop to ensure that all value logs are garbage collected
func (c *client) RunValueLogGC(discardRatio float64) error {
	for {
		err := c.db.RunValueLogGC(discardRatio)
		if errors.Is(err, badger.ErrNoRewrite) {
			return nil
		}
		if err != nil {
			return err
		}
	}
}

// DiskUsage is a container for holding disk usage of the badger client
type DiskUsage struct {
	LSMUsage      int64
	ValueLogUsage int64
}

// DiskUsage returns the disk usage of the badger client
func (c *client) GetDiskUsage() (DiskUsage, error) {
	if c.db == nil {
		return DiskUsage{}, errors.New("database not open")
	}

	lsmSize, valueLogSize := c.db.Size()
	return DiskUsage{
		LSMUsage:      lsmSize,
		ValueLogUsage: valueLogSize,
	}, nil
}

// GetOperationCounts returns the counts of operations performed by the client
func (c *client) GetOperationCounts() OperationCounts {
	return OperationCounts{
		Get:    c.opGet.Load(),
		Set:    c.opSet.Load(),
		Delete: c.opDelete.Load(),
	}
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

// badgerNopLogger is a nop logger for badger so we don't get output from badger if it tries to log anything
type badgerNopLogger struct {
	logger *zap.SugaredLogger
}

// Warningf logs a warning message
func (bnl *badgerNopLogger) Warningf(format string, v ...any) {
	bnl.logger.Warnf(format, v...)
}

// Debugf logs a debug message
func (bnl *badgerNopLogger) Debugf(format string, v ...any) {
	bnl.logger.Debugf(format, v...)
}

// Infof logs an info message
func (bnl *badgerNopLogger) Infof(format string, v ...any) {
	bnl.logger.Infof(format, v...)
}

// Errorf logs an error message
func (bnl *badgerNopLogger) Errorf(format string, v ...any) {
	bnl.logger.Errorf(format, v...)
}
