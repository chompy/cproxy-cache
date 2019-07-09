package ccache

import (
	"io/ioutil"
	"net/http"
	"regexp"
)

// EsiTag - data on esi tag
type EsiTag struct {
	URL      string
	Position int
}

// esiTagRegex - Regex for esi tag
const esiTagRegex = "<esi:include.*src=\\\"(.*)\\\".*>"

// ParseESI - take http response and parse out esi tags
func ParseESI(r *http.Response) (*http.Response, []EsiTag, error) {
	// compile regex
	regex, err := regexp.Compile(esiTagRegex)
	if err != nil {
		return nil, nil, err
	}
	// read response
	respBytes, err := HTTPResponseToBytes(r)
	if err != nil {
		return nil, nil, err
	}
	// match regex
	matches := regex.FindAllIndex(respBytes, -1)
	// itterate matches and create list of tags
	esiTagData := make([]EsiTag, 0)
	posOffset := 0
	for _, match := range matches {
		// extract url
		attrMatches := regex.FindAllSubmatch(respBytes[match[0]-posOffset:match[1]], -1)
		if len(attrMatches) < 1 || len(attrMatches[0]) < 2 {
			continue
		}
		urlAttr := string(attrMatches[0][1])
		if urlAttr == "" {
			continue
		}
		// remove tag from response
		lenBefore := len(respBytes)
		respBytes = append(respBytes[:match[0]-posOffset], respBytes[match[1]+1-posOffset:]...)
		// add tag data
		esiTagData = append(
			esiTagData,
			EsiTag{
				URL:      urlAttr,
				Position: match[0] - posOffset,
			},
		)
		posOffset += lenBefore - len(respBytes)
	}
	// create new output response
	outputResp, err := HTTPResponseFromBytes(respBytes)
	return outputResp, esiTagData, err
}

// ExpandESI - expand given esi tags in to response
func ExpandESI(r *http.Response, esiTags []EsiTag, subResps []*http.Response, cacheHandler *Handler) (*http.Response, error) {
	// read response
	respBytes, err := HTTPResponseToBytes(r)
	if err != nil {
		return nil, err
	}
	// itterate tags
	posOffset := 0
	for index, esiTag := range esiTags {
		// assume that sub resps are in same order as esi tags
		subResp := subResps[index]
		esiBodyBytes, err := ioutil.ReadAll(subResp.Body)
		if err != nil {
			return nil, err
		}
		subResp.Body.Close()
		// TODO this might could be improved apon
		// SEE https://github.com/golang/go/wiki/SliceTricks#insert
		respBytes = append(respBytes[:esiTag.Position+posOffset], append(esiBodyBytes, respBytes[esiTag.Position+posOffset:]...)...)
		posOffset += len(esiBodyBytes)
	}
	// rebuild response
	return HTTPResponseFromBytes(respBytes)
}
