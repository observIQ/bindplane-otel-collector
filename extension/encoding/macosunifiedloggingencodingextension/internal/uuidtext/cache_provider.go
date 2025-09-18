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

package uuidtext // import "github.com/observiq/bindplane-otel-collector/extension/encoding/macosunifiedloggingencodingextension/internal/uuidtext"

import (
	"sync"

	"github.com/observiq/bindplane-otel-collector/extension/encoding/macosunifiedloggingencodingextension/internal/sharedcache"
)

// CacheProvider handles caching of parsed UUID text and DSC data
type CacheProvider struct {
	uuidTextCache map[string]*UUIDText
	dscCache      map[string]*sharedcache.Strings
	mu            sync.RWMutex
}

// NewCacheProvider creates a new CacheProvider
func NewCacheProvider() *CacheProvider {
	return &CacheProvider{
		uuidTextCache: make(map[string]*UUIDText),
		dscCache:      make(map[string]*sharedcache.Strings),
	}
}

// CachedUUIDText returns the cached UUID text for the given UUID
func (c *CacheProvider) CachedUUIDText(uuid string) (*UUIDText, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	data, exists := c.uuidTextCache[uuid]
	return data, exists
}

// UpdateUUID updates the cached UUID text for the given UUID with actual data
func (c *CacheProvider) UpdateUUID(uuid string, uuid2 string, data *UUIDText) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Keep 30 UUID text files cached
	if len(c.uuidTextCache) > 30 {
		// Remove some old entries
		count := 0
		for k := range c.uuidTextCache {
			if count >= 5 {
				break
			}
			if k != uuid && k != uuid2 {
				delete(c.uuidTextCache, k)
				count++
			}
		}
	}
	c.uuidTextCache[uuid] = data
}

// CachedDSC returns the cached DSC for the given UUID
func (c *CacheProvider) CachedDSC(uuid string) (*sharedcache.Strings, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	data, exists := c.dscCache[uuid]
	return data, exists
}

// UpdateDSC updates the cached DSC for the given UUID with actual data
func (c *CacheProvider) UpdateDSC(uuid string, uuid2 string, data *sharedcache.Strings) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Keep only 2 DSC files cached (they're large ~30-150MB)
	for len(c.dscCache) > 2 {
		for k := range c.dscCache {
			if k != uuid && k != uuid2 {
				delete(c.dscCache, k)
				break
			}
		}
	}
	c.dscCache[uuid] = data
}
