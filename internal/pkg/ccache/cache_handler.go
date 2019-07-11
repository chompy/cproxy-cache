package ccache

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/pquerna/cachecontrol/cacheobject"
)

// Handler - main cache handler
type Handler struct {
	Config             Config
	CacheItems         []Item
	lastClean          time.Time
	locked             bool
	subRequestCallback func(req *http.Request) (*http.Response, error)
}

// NewHandler - create new cache handler
func NewHandler(config Config, subRequestCallback func(req *http.Request) (*http.Response, error)) Handler {
	handler := Handler{
		Config:             config,
		subRequestCallback: subRequestCallback,
	}
	handler.Clear()
	return handler
}

// fetchItem - fetch cache item with key
func (b *Handler) fetchItem(key string) *Item {
	if key == "" {
		return nil
	}
	for index := range b.CacheItems {
		if b.CacheItems[index].Key == key {
			if b.CacheItems[index].HasExpired() {
				continue
			}
			return &b.CacheItems[index]
		}
	}
	return nil
}

// Fetch - fetch cache item from request
func (b *Handler) Fetch(r *http.Request) *Item {
	// look for a private key first
	item := b.fetchItem(PrivateKeyFromRequest(r, &b.Config))
	if item != nil {
		return item
	}
	// fallback to public key
	return b.fetchItem(PublicKeyFromRequest(r, &b.Config))
}

// Store - store response if cachable
func (b *Handler) Store(resp *http.Response) (*Item, error) {
	// only cache certain request/response types
	if resp.Request.Method != "GET" || resp.StatusCode < 200 || resp.StatusCode > 299 {
		return nil, nil
	}
	// parse cache control header
	cacheControl, err := cacheobject.ParseResponseCacheControl(resp.Header.Get("Cache-Control"))
	if err != nil {
		return nil, err
	}
	// no cache
	if cacheControl.NoCachePresent || cacheControl.NoStore {
		return nil, nil
	}
	// configured to not cache private responses
	if cacheControl.PrivatePresent && !b.Config.CachePrivate {
		return nil, nil
	}
	// too large to cache
	if resp.ContentLength > int64(b.Config.ResponseMaxSize) {
		return nil, nil
	}
	// check max age, if zero, don't cache
	cacheMaxAge := int32(cacheControl.MaxAge)
	if cacheControl.SMaxAge > 0 {
		cacheMaxAge = int32(cacheControl.SMaxAge)
	}
	if cacheMaxAge == 0 {
		return nil, nil
	}
	// already exists?
	cacheItem := b.Fetch(resp.Request)
	if cacheItem != nil {
		return cacheItem, nil
	}
	// create cache item
	newCacheItem, err := ItemFromResponse(resp, &b.Config)
	if err != nil {
		return nil, err
	}
	// set max age
	newCacheItem.MaxAge = cacheMaxAge
	// store cache item
	b.CacheItems = append(b.CacheItems, newCacheItem)
	if err != nil {
		return nil, err
	}
	return &b.CacheItems[len(b.CacheItems)-1], nil
}

// clearItemIndex - clear a cache item from its index
func (b *Handler) clearItemIndex(index int) {
	for b.locked {
	}
	if index < 0 || index > len(b.CacheItems)-1 {
		b.locked = false
		return
	}
	b.locked = true
	b.CacheItems[index].Clear()
	b.CacheItems[index] = b.CacheItems[len(b.CacheItems)-1]
	b.CacheItems = b.CacheItems[:len(b.CacheItems)-1]
	b.locked = false
}

// Invalidate - remove matching items from cache
func (b *Handler) Invalidate(req *http.Request) {
	switch req.Method {
	case "PURGE", "BAN":
		{
			// TODO need debug logging
			//log.Println("--", r.Method, r.URL.Path, r.Header)
			// remove a object found at given url path
			// TODO, focus on header based invalidate for now
			/*for index, item := range b.CacheItems {
				if r.URL.Path == item.Path {
					item.LogAction("invalidate", "REASON = path match")
					b.clearItemIndex(index)
					break
				}
			}*/
			// remove any object which match any of the ban headers
			// retrieve ban header key+values
			invalidateHeaderValues := map[string]string{}
			for _, key := range b.Config.InvalidateHeaders {
				reqVal := req.Header.Get(key)
				switch key {
				case "Xkey":
					{
						if reqVal == "" {
							reqVal = req.Header.Get("Key")
						}
					}
				}
				if reqVal != "" {
					invalidateHeaderValues[key] = reqVal
				}
			}
			// itterate items in cache
			for index, item := range b.CacheItems {
				hasInvalidate := false
				// check invalidate headers
				if len(invalidateHeaderValues) > 0 {
					invalidateHeaderMatchCount := 0
					for key, reqVal := range invalidateHeaderValues {
						for _, cacheVal := range item.InvalidateHeaders[key] {
							hasMatch := false
							switch key {
							case "Xkey":
								{
									if strings.Contains(cacheVal, reqVal) {
										hasMatch = true
									}
									break
								}
							default:
								{
									if WildcardCompare(cacheVal, reqVal) {
										hasMatch = true
										break
									}
									regex, err := regexp.Compile(reqVal)
									if err == nil && regex.MatchString(cacheVal) {
										hasMatch = true
									}
									break
								}
							}
							if hasMatch {
								invalidateHeaderMatchCount++
							}
						}
					}
					if invalidateHeaderMatchCount == len(invalidateHeaderValues) {
						hasInvalidate = true
						item.LogAction("invalidate", "REASON = header match")
					}
				}
				// perform ban
				if hasInvalidate {
					b.clearItemIndex(index)
					// check for more invalidations
					b.Invalidate(req)
					return
				}
			}
		}
	}
}

// OnRequest - handle incomming request
func (b *Handler) OnRequest(req *http.Request) (*http.Response, error) {
	// add header to request for ESI
	if b.Config.UseESI {
		req.Header.Add("Surrogate-Capability", "content=ESI/1.0")
	}
	// handle request
	switch req.Method {
	case "BAN", "PURGE":
		{
			body := ""
			resp := &http.Response{
				Status:        "200 OK",
				StatusCode:    200,
				Proto:         "HTTP/1.1",
				ProtoMajor:    1,
				ProtoMinor:    1,
				Body:          ioutil.NopCloser(bytes.NewBufferString(body)),
				ContentLength: int64(len(body)),
				Request:       req,
				Header:        make(http.Header, 0),
			}
			resp.Header.Set("Content-Type", "text/plain")
			// ensure valid host
			// must be localhost, TODO extend this?
			if req.RemoteAddr != ":0" {
				resp.Status = "405 Not Allowed"
				resp.StatusCode = 405
				return resp, nil
			}
			// invalidate
			b.Invalidate(req)
			return resp, nil
		}
	case http.MethodGet:
		{
			// clean cache
			b.Clean()
			// get cache item
			cacheItem := b.Fetch(req)
			// none exist, no cache
			if cacheItem == nil {
				return nil, nil
			}
			// get response from cache
			resp, err := cacheItem.GetResponse()
			if err != nil {
				return nil, err
			}
			resp.Request = req
			// update cache hit count and set cache response headers
			cacheItem.Hits++
			cacheItem.LastHit = time.Now()
			resp.Header.Set("X-Cache", "HIT")
			resp.Header.Set("X-Cache-Count", strconv.Itoa(cacheItem.Hits))
			cacheItem.LogAction("fetch", fmt.Sprintf("COUNT = %d", cacheItem.Hits))
			// return response with ESI tags expanded
			return ExpandESI(resp, b.subRequestCallback)
		}
	}
	return nil, nil
}

// OnResponse - handle outgoing response
func (b *Handler) OnResponse(resp *http.Response) (*http.Response, error) {
	// need request
	if resp.Request == nil {
		return resp, nil
	}
	// store response in cache if able
	cacheItem, err := b.Store(resp)
	if err != nil {
		return nil, err
	}
	if cacheItem == nil {
		return resp, nil
	}
	// retrieve stored response for output
	req := resp.Request
	resp, err = cacheItem.GetResponse()
	if err != nil {
		return nil, err
	}
	resp.Request = req
	// set cache response headers
	resp.Header.Set("X-Cache", "MISS")
	resp.Header.Set("X-Cache-Count", "0")
	// expand ESI into final response
	return ExpandESI(resp, b.subRequestCallback)
}

// Clear - clear all cache items
func (b *Handler) Clear() {
	os.RemoveAll(b.Config.CacheFilePath)
	os.MkdirAll(b.Config.CacheFilePath, 0770)
	b.CacheItems = make([]Item, 0)
	b.lastClean = time.Now()
	b.locked = false
}

// Clean - clean up cache items
func (b *Handler) Clean() {
	// time to clean?
	if time.Now().Add(time.Duration(-b.Config.CleanInterval) * time.Second).Before(b.lastClean) {
		return
	}
	log.Println("CACHE :: CLEAN")
	// clear expired
	hasExpired := true
	for hasExpired {
		hasExpired = false
		for index := range b.CacheItems {
			if b.CacheItems[index].HasExpired() {
				b.CacheItems[index].LogAction("invalidate", "REASON = max age expired")
				b.clearItemIndex(index)
				hasExpired = true
				break
			}
		}
	}
	// split cache storage in to different pools for each cache type (public/private)
	// and for each storage handler (memory/file)
	cacheTypes := []string{CacheItemPublic, CacheItemPrivate}
	for cacheTypeIndex, cacheType := range cacheTypes {
		for _, cacheStorageHandler := range b.Config.CacheStorageHandlers {
			// check size of pool
			cacheSize := int64(0)
			for index := range b.CacheItems {
				if b.CacheItems[index].Type == cacheType && b.CacheItems[index].GetStorageType() == cacheStorageHandler {
					cacheSize += b.CacheItems[index].Size
				}
			}
			// cache reached max size, clear oldest items
			for cacheSize > int64(b.Config.CacheMaxSize[cacheType][cacheStorageHandler]) {
				// find oldest item
				oldestItemIndex := -1
				for index, item := range b.CacheItems {
					if oldestItemIndex < 0 || item.LastHit.Before(b.CacheItems[oldestItemIndex].LastHit) {
						oldestItemIndex = index
					}
				}
				// move oldest item to next storage handler OR delete if last storage handler
				cacheSize -= b.CacheItems[oldestItemIndex].Size
				if cacheTypeIndex+1 < len(cacheTypes) {
					b.CacheItems[oldestItemIndex].MoveStorage(b.Config.CacheStorageHandlers[cacheTypeIndex+1], &b.Config)
				}
				// clear from cache
				b.clearItemIndex(oldestItemIndex)
			}
		}
	}
	b.lastClean = time.Now()
}
