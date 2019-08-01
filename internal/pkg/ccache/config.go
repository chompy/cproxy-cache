/*
This file is part of CProxy-Cache.

CProxy-Cache is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

CProxy-Cache is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with CProxy-Cache.  If not, see <https://www.gnu.org/licenses/>.
*/

package ccache

// VersionNo - current version of this extension
const VersionNo = 1

// Config - cache configuration
type Config struct {
	CacheStorageHandlers []string                  `json:"cache_storage_handlers"` // list of cache storage handlers to use (in order)
	CacheFilePath        string                    `json:"cache_file_path"`        // file path to file system cache
	CacheMaxSize         map[string]map[string]int `json:"cache_max_size"`         // max size for each cache type (public/private) and each cache storage handler
	ResponseMaxSize      int                       `json:"response_max_size"`      // max size allowed to cache a response
	CleanInterval        int                       `json:"clean_interval"`         // interval in seconds of when to performance cache clean up`
	VaryHeaders          []string                  `json:"vary_headers"`           // headers that should be used to calculate cache keys
	InvalidateHeaders    []string                  `json:"invalidate_headers"`     // list of headers to use for cache ban/purge requests
	CachePrivate         bool                      `json:"enable_private_cache"`   // whether or not to cache private content (cache-control: private)
	VaryCookies          []string                  `json:"vary_cookies"`           // list of cookies to use to vary private cache
	UseESI               bool                      `json:"enable_esi"`             // whether or not to handle ESI tags
}

// GetDefaultConfig - get default configuration
func GetDefaultConfig() Config {
	return Config{
		CacheStorageHandlers: []string{CacheStorageMemory, CacheStorageFile},
		CacheFilePath:        "/tmp/cproxy-cache/",
		CacheMaxSize: map[string]map[string]int{
			CacheItemPublic: {
				CacheStorageFile:   1024 * 1024 * 500, // 500MB
				CacheStorageMemory: 1024 * 1024 * 50,  // 50MB
			},
			CacheItemPrivate: {
				CacheStorageFile:   1024 * 1024 * 100, // 100MB
				CacheStorageMemory: 1024 * 1024 * 10,  // 10MB
			},
		},
		ResponseMaxSize:   1024 * 1024, // 1MB
		CleanInterval:     300,         // 5 minutes
		VaryHeaders:       []string{},
		InvalidateHeaders: []string{"X-Location-Id", "X-User-Hash", "X-Installion-Id", "X-Site-Name", "Xkey"},
		CachePrivate:      true,
		VaryCookies:       []string{"eZSESSID*", "PHPSESSID*"},
		UseESI:            true,
	}
}
