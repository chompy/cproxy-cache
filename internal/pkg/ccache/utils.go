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
