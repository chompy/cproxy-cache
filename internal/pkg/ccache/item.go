package ccache

import (
	"crypto/md5"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/pquerna/cachecontrol/cacheobject"
)

// CacheItemPublic - flag, denotes cache item is public cache
const CacheItemPublic = "public"

// CacheItemPrivate - flag, denotes cache item is private cache
const CacheItemPrivate = "private"

// Item - cached item
type Item struct {
	Type              string
	Key               string
	Hits              int
	Size              int64
	Created           time.Time
	LastHit           time.Time
	MaxAge            int32
	Path              string
	InvalidateHeaders map[string][]string
	EsiTags           []EsiTag
	storage           Storage
}

// PublicKeyFromRequest - generate public cache key from request
func PublicKeyFromRequest(r *http.Request, config *Config) string {
	h := md5.New()
	io.WriteString(h, r.Method)
	io.WriteString(h, r.URL.Path)
	io.WriteString(h, r.URL.Query().Encode())
	for headerName, headerValues := range r.Header {
		for _, varyHeaderName := range config.VaryHeaders {
			if WildcardCompare(headerName, varyHeaderName) {
				io.WriteString(h, headerName)
				for _, headerValue := range headerValues {
					io.WriteString(h, headerValue)
				}
				break
			}
		}
	}
	return fmt.Sprintf("%x", h.Sum(nil))
}

// PrivateKeyFromRequest - generate private cache key from request
func PrivateKeyFromRequest(r *http.Request, config *Config) string {
	h := md5.New()
	io.WriteString(h, PublicKeyFromRequest(r, config))
	io.WriteString(h, r.RemoteAddr)
	io.WriteString(h, r.UserAgent())
	for _, cookie := range r.Cookies() {
		for _, cookieNameMatch := range config.VaryCookies {
			if WildcardCompare(cookie.Name, cookieNameMatch) {
				io.WriteString(h, cookie.Name)
				io.WriteString(h, cookie.Value)
				break
			}
		}
	}
	return fmt.Sprintf("%x", h.Sum(nil))
}

// ItemFromResponse - create cache item from response
func ItemFromResponse(resp *http.Response, config *Config) (Item, error) {
	// parse cache control header
	cacheControl, err := cacheobject.ParseResponseCacheControl(resp.Header.Get("Cache-Control"))
	if err != nil {
		return Item{}, err
	}
	// determine cache item type
	itemType := CacheItemPublic
	if cacheControl.PrivatePresent {
		itemType = CacheItemPrivate
	}
	// create key for cache item
	key := ""
	switch itemType {
	case CacheItemPublic:
		{
			key = PublicKeyFromRequest(resp.Request, config)
			break
		}
	case CacheItemPrivate:
		{
			key = PrivateKeyFromRequest(resp.Request, config)
			break
		}
	default:
		{
			return Item{}, nil
		}
	}
	// set custom vars (used for BAN/PURGE)
	invalidateHeaders := map[string][]string{}
	for _, headerName := range config.InvalidateHeaders {
		invalidateHeaders[headerName] = resp.Header[headerName]
	}
	// get cache storage handler
	if len(config.CacheStorageHandlers) == 0 {
		return Item{}, errors.New("no storage handler provided")
	}
	storage := GetStorageHandler(config.CacheStorageHandlers[0])
	if storage == nil {
		return Item{}, fmt.Errorf("could not find storage handler '%s'", config.CacheStorageHandlers[0])
	}
	// create cache item struct
	item := Item{
		Type:              itemType,
		Key:               key,
		Path:              resp.Request.URL.Path,
		Hits:              0,
		Size:              0,
		Created:           time.Now(),
		InvalidateHeaders: invalidateHeaders,
		storage:           storage,
	}
	// log
	item.LogAction("create", "-")
	// init storage
	item.storage.Init(key, config)
	// parse esi
	resp, esiTags, err := ParseESI(resp)
	if err != nil {
		return item, err
	}
	item.EsiTags = esiTags
	// store response
	err = item.storage.StoreResponse(resp)
	if err != nil {
		return item, err
	}
	// get size of response in storage
	item.Size, err = item.storage.GetSize()
	return item, err
}

// GetResponse - convert cache item in to http response
func (i *Item) GetResponse() (*http.Response, error) {
	resp, err := i.storage.FetchResponse()
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// GetStorageType - get storage type name
func (i *Item) GetStorageType() string {
	return i.storage.GetTypeName()
}

// MoveStorage - convert current storage to given new storage
func (i *Item) MoveStorage(name string, config *Config) error {
	// log action
	i.LogAction("move", fmt.Sprintf("Move storage to '%s'", name))
	// init new storage
	newStorage := GetStorageHandler(name)
	if newStorage == nil {
		return fmt.Errorf("could not find storage handler '%s'", name)
	}
	newStorage.Init(i.Key, config)
	// fetch response from old storage
	resp, err := i.storage.FetchResponse()
	if err != nil {
		return err
	}
	// store response in new storage
	err = newStorage.StoreResponse(resp)
	if err != nil {
		return err
	}
	// clear old storage
	err = i.storage.Delete()
	if err != nil {
		return err
	}
	i.storage = newStorage
	return nil
}

// HasExpired - check if this cache item has expired
func (i *Item) HasExpired() bool {
	// check max age
	now := time.Now()
	expireTime := i.Created.Add(time.Duration(i.MaxAge) * time.Second)
	if !now.Before(expireTime) {
		return true
	}
	return false
}

// Clear - delete this cache item
func (i *Item) Clear() {
	i.storage.Delete()
	i.Size = 0
}

// LogAction - log action taken against cache item
func (i *Item) LogAction(action string, desc string) {
	log.Printf("CACHE :: %s :: %s - %s :: %s", strings.ToUpper(action), i.Key, i.Path, desc)
}
