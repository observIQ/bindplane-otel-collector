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
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"go.etcd.io/bbolt"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/extension/xextension/storage"
	"go.uber.org/zap"
)

const (
	cacheBucketName = "lookup_cache"
)

var (
	errCacheNotEnabled = errors.New("cache is not enabled")
)

// cacheEntry represents a cached lookup result with timestamp
type cacheEntry struct {
	Data      map[string]string `json:"data"`
	Timestamp int64             `json:"timestamp"`
}

// LookupCache wraps a LookupSource with caching capability
type LookupCache struct {
	source  LookupSource
	db      *bbolt.DB
	storage storage.Client
	ttl     time.Duration
	enabled bool
	logger  *zap.Logger
}

// NewLookupCache creates a new LookupCache
func NewLookupCache(
	ctx context.Context,
	source LookupSource,
	ttl time.Duration,
	enabled bool,
	storageID *component.ID,
	host component.Host,
	componentID component.ID,
	logger *zap.Logger,
) (*LookupCache, error) {
	cache := &LookupCache{
		source:  source,
		ttl:     ttl,
		enabled: enabled,
		logger:  logger,
	}

	if !enabled {
		return cache, nil
	}

	// Use shared storage if storage ID is provided
	if storageID != nil {
		client, err := getStorageClient(ctx, host, *storageID, componentID)
		if err != nil {
			return nil, fmt.Errorf("failed to get storage client: %w", err)
		}
		cache.storage = client
	} else {
		// Use processor-local bbolt database
		db, err := openLocalDB(componentID)
		if err != nil {
			return nil, fmt.Errorf("failed to open local cache database: %w", err)
		}
		cache.db = db

		// Create bucket if it doesn't exist
		err = db.Update(func(tx *bbolt.Tx) error {
			_, err := tx.CreateBucketIfNotExists([]byte(cacheBucketName))
			return err
		})
		if err != nil {
			_ = db.Close()
			return nil, fmt.Errorf("failed to create cache bucket: %w", err)
		}
	}

	return cache, nil
}

// Lookup checks the cache first, then falls back to the source
func (c *LookupCache) Lookup(key string) (map[string]string, error) {
	if !c.enabled {
		return c.source.Lookup(key)
	}

	// Check cache first
	cachedData, found, err := c.get(key)
	if err != nil {
		c.logger.Debug("cache lookup error, falling back to source", zap.Error(err))
	} else if found {
		c.logger.Debug("cache hit", zap.String("key", key))
		return cachedData, nil
	}

	// Cache miss, lookup from source
	c.logger.Debug("cache miss", zap.String("key", key))
	data, err := c.source.Lookup(key)
	if err != nil {
		return nil, err
	}

	// Store in cache
	if storeErr := c.set(key, data); storeErr != nil {
		c.logger.Debug("failed to cache result", zap.Error(storeErr))
	}

	return data, nil
}

// Load initializes the source
func (c *LookupCache) Load() error {
	return c.source.Load()
}

// Close cleans up resources
func (c *LookupCache) Close() error {
	var errs []error

	if c.source != nil {
		if err := c.source.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close source: %w", err))
		}
	}

	if c.db != nil {
		if err := c.db.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close local cache db: %w", err))
		}
	}

	if c.storage != nil {
		if err := c.storage.Close(context.Background()); err != nil {
			errs = append(errs, fmt.Errorf("failed to close storage client: %w", err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("errors during close: %v", errs)
	}

	return nil
}

// get retrieves a value from cache and checks if it's expired
func (c *LookupCache) get(key string) (map[string]string, bool, error) {
	cacheKey := fmt.Sprintf("lookup:%s", key)
	var data []byte
	var err error

	if c.storage != nil {
		data, err = c.storage.Get(context.Background(), cacheKey)
		if err != nil {
			return nil, false, err
		}
		if data == nil {
			return nil, false, nil
		}
	} else if c.db != nil {
		err = c.db.View(func(tx *bbolt.Tx) error {
			bucket := tx.Bucket([]byte(cacheBucketName))
			if bucket == nil {
				return nil
			}
			data = bucket.Get([]byte(cacheKey))
			return nil
		})
		if err != nil {
			return nil, false, err
		}
		if data == nil {
			return nil, false, nil
		}
	} else {
		return nil, false, errCacheNotEnabled
	}

	// Decode cache entry
	var entry cacheEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		return nil, false, fmt.Errorf("failed to unmarshal cache entry: %w", err)
	}

	// Check if expired
	if time.Now().Unix()-entry.Timestamp > int64(c.ttl.Seconds()) {
		c.logger.Debug("cache entry expired", zap.String("key", key))
		return nil, false, nil
	}

	return entry.Data, true, nil
}

// set stores a value in cache with current timestamp
func (c *LookupCache) set(key string, data map[string]string) error {
	cacheKey := fmt.Sprintf("lookup:%s", key)
	entry := cacheEntry{
		Data:      data,
		Timestamp: time.Now().Unix(),
	}

	entryData, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("failed to marshal cache entry: %w", err)
	}

	if c.storage != nil {
		return c.storage.Set(context.Background(), cacheKey, entryData)
	} else if c.db != nil {
		return c.db.Update(func(tx *bbolt.Tx) error {
			bucket := tx.Bucket([]byte(cacheBucketName))
			if bucket == nil {
				return errors.New("cache bucket not found")
			}
			return bucket.Put([]byte(cacheKey), entryData)
		})
	}

	return errCacheNotEnabled
}

// openLocalDB opens a processor-local bbolt database
func openLocalDB(componentID component.ID) (*bbolt.DB, error) {
	// Create cache directory in temp
	cacheDir := filepath.Join(os.TempDir(), "otel-lookup-cache")
	if err := os.MkdirAll(cacheDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create cache directory: %w", err)
	}

	// Create database file based on component ID
	dbPath := filepath.Join(cacheDir, fmt.Sprintf("%s-%s.db", componentID.Type(), componentID.Name()))
	db, err := bbolt.Open(dbPath, 0600, &bbolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		return nil, fmt.Errorf("failed to open bbolt database: %w", err)
	}

	return db, nil
}

// getStorageClient gets a storage client from the host
func getStorageClient(ctx context.Context, host component.Host, storageID component.ID, componentID component.ID) (storage.Client, error) {
	extension, ok := host.GetExtensions()[storageID]
	if !ok {
		return nil, fmt.Errorf("storage extension '%s' not found", storageID)
	}

	storageExtension, ok := extension.(storage.Extension)
	if !ok {
		return nil, fmt.Errorf("extension '%s' is not a storage extension", storageID)
	}

	client, err := storageExtension.GetClient(ctx, component.KindProcessor, componentID, "")
	if err != nil {
		return nil, fmt.Errorf("failed to get storage client: %w", err)
	}

	return client, nil
}
