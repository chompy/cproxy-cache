package ccache

import (
	"bufio"
	"bytes"
	"net/http"
	"regexp"
	"strings"
)

// HTTPResponseToBytes - convert http response to bytes
func HTTPResponseToBytes(r *http.Response) ([]byte, error) {
	buf := bytes.NewBuffer(nil)
	bufW := bufio.NewWriter(buf)
	err := r.Write(bufW)
	if err != nil {
		return nil, err
	}
	bufW.Flush()
	r.Body.Close()
	return buf.Bytes(), nil
}

// HTTPResponseFromBytes - convert bytes to http response
func HTTPResponseFromBytes(b []byte) (*http.Response, error) {
	return http.ReadResponse(
		bufio.NewReader(bytes.NewReader(b)),
		nil,
	)
}

// wildcardMatchCharacter - character to use as the wildcard character
const wildcardMatchCharacter = "*"

// WildcardCompare - test if original string matches test string with wildcard
func WildcardCompare(original string, test string) bool {
	test = regexp.QuoteMeta(test)
	test = strings.Replace(test, "\\*", ".*", -1)
	regex, err := regexp.Compile(test + "$")
	if err != nil {
		return false
	}
	return regex.MatchString(original)
}
