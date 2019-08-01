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

package main

import (
	"encoding/json"
	"log"
	"net/http"

	ccache "./internal/pkg/ccache"
)

// cacheHandler - cache handler
var cacheHandler ccache.Handler

// GetName - get name of this extension
func GetName() string {
	return "Cproxy-Cache"
}

// OnLoad - load the extension
func OnLoad(subRequestCallback func(*http.Request) (*http.Response, error), rawConfig []byte) error {
	log.Printf("CACHE :: Init %s v%.2f", GetName(), ccache.VersionNo/100.0)
	// load config
	config := ccache.GetDefaultConfig()
	if rawConfig != nil && len(rawConfig) > 0 {
		err := json.Unmarshal(rawConfig, &config)
		if err != nil {
			return err
		}
	}
	// init cache handler
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
