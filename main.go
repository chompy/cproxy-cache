package main

import (
	"log"
	"net/http"

	ccache "./internal/pkg/ccache"
)

// cacheHandler - cache handler
var cacheHandler ccache.Handler

// OnLoad - load the extension
func OnLoad(subRequestCallback func(*http.Request) (*http.Response, error)) error {
	log.Printf("CACHE :: Init CCache v%.2f", ccache.VersionNo/100.0)
	config := ccache.GetDefaultConfig()
	cacheHandler = ccache.NewHandler(config, subRequestCallback)
	return nil
}

// OnUnload - unload extension
func OnUnload() {
	cacheHandler.Clear()
}

// OnRequest - request event
func OnRequest(req *http.Request) (*http.Response, error) {
	log.Println("CACHE :: OnRequest")
	return cacheHandler.OnRequest(req)
}

// OnResponse - response event
func OnResponse(resp *http.Response) (*http.Response, error) {
	log.Println("CACHE :: OnResponse")
	return cacheHandler.OnResponse(resp)
}

// not used
func main() {
	return
}
