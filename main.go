package main

import (
	"log"
	"net/http"

	ccache "./internal/pkg/ccache"
)

// cacheHandler - cache handler
var cacheHandler *ccache.Handler

// OnLoad - extension load
func OnLoad() error {
	log.Printf("CACHE :: Init CCache v%.2f", ccache.VersionNo/100.0)
	config := ccache.GetDefaultConfig()
	cacheHandler = ccache.NewHandler(config)
	return nil
}

// OnUnload - extension unload
func OnUnload() error {
	cacheHandler.Clear()
	return nil
}

// OnRequest - recieve request from cproxy
func OnRequest(request *http.Request) (*http.Response, error) {
	return cacheHandler.HandleRequest(request)
}

// OnCollectSubRequests - recieve response from cproxy so that sub requests can be determined
func OnCollectSubRequests(resp *http.Response) ([]*http.Request, error) {
	// must have request with response
	if resp.Request == nil {
		return nil, nil
	}
	// fetch esi tags
	cacheItem := cacheHandler.Fetch(resp.Request)
	if cacheItem == nil {
		var err error
		// store response so esi tag can be processed
		cacheItem, err = cacheHandler.Store(resp)
		if err != nil {
			return nil, err
		}
		if cacheItem == nil {
			return nil, nil
		}
	}
	// create esi requests
	esiReqs := make([]*http.Request, 0)
	for _, esiTag := range cacheItem.EsiTags {
		esiReq, err := http.NewRequest(
			http.MethodGet,
			resp.Request.URL.Scheme+"://"+resp.Request.URL.Host+esiTag.URL,
			nil,
		)
		if err != nil {
			return nil, err
		}
		esiReq = esiReq.WithContext(resp.Request.Context())
		esiReq.Header = resp.Request.Header
		esiReqs = append(esiReqs, esiReq)
	}
	return esiReqs, nil
}

// OnResponse - recieve response from cproxy
func OnResponse(resp *http.Response, subResps []*http.Response) (*http.Response, error) {
	cacheItem := cacheHandler.Fetch(resp.Request)
	if cacheItem == nil {
		return resp, nil
	}
	return cacheHandler.GetESIResponse(cacheItem, subResps)
}

// not used
func main() {
	return
}
