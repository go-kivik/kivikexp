// Licensed under the Apache License, Version 2.0 (the "License"); you may not
// use this file except in compliance with the License. You may obtain a copy of
// the License at
//
//  http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
// WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the
// License for the specific language governing permissions and limitations under
// the License.

package cmd

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strings"
	"testing"

	"gitlab.com/flimzy/testy"

	"github.com/go-kivik/xkivik/v4/cmd/kouchctl/errors"
)

func Test_put_security_RunE(t *testing.T) {
	tests := testy.NewTable()

	tests.Add("missing database", func(t *testing.T) interface{} {
		return cmdTest{
			args:   []string{"put", "security", "http://example.com/foo/"},
			status: errors.ErrUsage,
		}
	})
	tests.Add("success", func(t *testing.T) interface{} {
		s := testy.ServeResponseValidator(t, &http.Response{
			StatusCode: http.StatusOK,
			Header: http.Header{
				"Content-Type": []string{"application/json"},
			},
			Body: ioutil.NopCloser(strings.NewReader(`"old"`)),
		}, func(t *testing.T, req *http.Request) {
			if req.Method != http.MethodPut {
				t.Errorf("Unexpected method: %v", req.Method)
			}
			want := json.RawMessage(`{"admins":{},"members":{}}`)
			if d := testy.DiffAsJSON(want, req.Body); d != nil {
				t.Errorf("Unexpected request body: %s", d)
			}
			if req.URL.Path != "/db/_security" {
				t.Errorf("unexpected path: %s", req.URL.Path)
			}
		})

		return cmdTest{
			args: []string{"put", "security", s.URL + "/db", "--data", "{}"},
		}
	})

	tests.Run(t, func(t *testing.T, tt cmdTest) {
		tt.Test(t)
	})
}
