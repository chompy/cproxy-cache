package ccache

import (
	"io/ioutil"
	"net/http"
	"regexp"
)

// esiTagRegex - Regex for esi tag
const esiTagRegex = `(?U)<esi:include.*src=\"(.*)\".*>`

// ExpandESI - take http response and replace esi tags
func ExpandESI(resp *http.Response, subRequestCallback func(req *http.Request) (*http.Response, error)) (*http.Response, error) {
	// must have request attached to response
	if resp.Request == nil {
		return resp, nil
	}
	// compile regex
	regex, err := regexp.Compile(esiTagRegex)
	if err != nil {
		return nil, err
	}
	// read response
	respBytes, err := HTTPResponseToBytes(resp)
	if err != nil {
		return nil, err
	}
	// match regex
	matches := regex.FindAllIndex(respBytes, -1)
	// itterate matches and process esi request
	posOffset := 0
	for _, match := range matches {

		// extract url
		attrMatches := regex.FindAllSubmatch(respBytes[match[0]+posOffset:match[1]+posOffset], -1)
		if len(attrMatches) < 1 || len(attrMatches[0]) < 2 {
			continue
		}
		urlAttr := string(attrMatches[0][1])
		if urlAttr == "" {
			continue
		}

		// make sub request for esi data
		// TODO be smarter about processing the ESI url
		esiReq, err := http.NewRequest(
			http.MethodGet,
			resp.Request.URL.Scheme+"://"+resp.Request.URL.Host+urlAttr,
			nil,
		)
		if err != nil {
			return nil, err
		}
		esiReq = esiReq.WithContext(resp.Request.Context())
		esiReq.Header = resp.Request.Header

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
			respBytes[:match[0]+posOffset],
			append(
				esiBodyBytes,
				respBytes[match[1]+posOffset:]...,
			)...,
		)
		posOffset += -(match[1] - match[0]) + len(esiBodyBytes)
	}
	// create new output response
	outputResp, err := HTTPResponseFromBytes(respBytes)
	return outputResp, err
}
