package ccache

import (
	"io/ioutil"
	"net/http"
	"regexp"
)

// esiTagRegex - Regex for esi tag
const esiTagRegex = `(?U)<esi:include.*src=\"(.*)\".*>`

// esiTagRegexCompiled - Compiled esi regex
var esiTagRegexCompiled *regexp.Regexp

// EsiTag - data about ESI tag
type EsiTag struct {
	URL      string // url for ESI request
	Position int    // position in original response of ESI tag
}

// ParseESI - parse esi tags from response
func ParseESI(resp *http.Response) (*http.Response, []EsiTag, error) {

	// save original request
	req := resp.Request
	// compile regex
	if esiTagRegexCompiled == nil {
		var err error
		esiTagRegexCompiled, err = regexp.Compile(esiTagRegex)
		if err != nil {
			return nil, nil, err
		}
	}
	// read response
	respBytes, err := HTTPResponseToBytes(resp)
	if err != nil {
		return nil, nil, err
	}
	esiTags := make([]EsiTag, 0)
	// match regex
	posOffset := 0
	matches := esiTagRegexCompiled.FindAllIndex(respBytes, -1)
	for _, match := range matches {
		// extract url
		attrMatches := esiTagRegexCompiled.FindAllSubmatch(respBytes[match[0]-posOffset:match[1]-posOffset], -1)
		if len(attrMatches) < 1 || len(attrMatches[0]) < 2 {
			continue
		}
		urlAttr := string(attrMatches[0][1])
		if urlAttr == "" {
			continue
		}
		// remove esi tag
		respBytes = append(
			respBytes[:match[0]-posOffset],
			respBytes[match[1]-posOffset:]...,
		)
		// store esi tag data
		esiTag := EsiTag{
			URL:      urlAttr,
			Position: match[0] - posOffset,
		}
		esiTags = append(esiTags, esiTag)
		// increase offset
		posOffset += match[1] - match[0]
	}
	// create new output response
	outputResp, err := HTTPResponseFromBytes(respBytes)
	outputResp.Request = req
	return outputResp, esiTags, nil

}

// ExpandESI - take http response and replace esi tags
func ExpandESI(resp *http.Response, esiTags []EsiTag, subRequestCallback func(req *http.Request) (*http.Response, error)) (*http.Response, error) {

	// must have request attached to response
	if resp.Request == nil {
		return resp, nil
	}
	req := resp.Request
	// read response
	respBytes, err := HTTPResponseToBytes(resp)
	if err != nil {
		return nil, err
	}
	// itterate tags and
	posOffset := 0
	for _, esiTag := range esiTags {
		// make sub request for esi data
		// TODO be smarter about processing the ESI url
		esiReq, err := http.NewRequest(
			http.MethodGet,
			req.URL.Scheme+"://"+req.URL.Host+esiTag.URL,
			nil,
		)
		if err != nil {
			return nil, err
		}
		esiReq = esiReq.WithContext(req.Context())
		esiReq.Header = req.Header
		// perform sub request
		esiResp, err := subRequestCallback(esiReq)
		if err != nil {
			return nil, err
		}
		if esiResp == nil {
			continue
		}
		// get esi response body
		esiBodyBytes, err := ioutil.ReadAll(esiResp.Body)
		if err != nil {
			return nil, err
		}
		esiResp.Body.Close()
		// replace esi tag with esi response
		respBytes = append(
			respBytes[:esiTag.Position+posOffset],
			append(
				esiBodyBytes,
				respBytes[esiTag.Position+posOffset:]...,
			)...,
		)
		posOffset += len(esiBodyBytes)
	}

	// create new output response
	outputResp, err := HTTPResponseFromBytes(respBytes)
	outputResp.Request = req
	return outputResp, err

}
