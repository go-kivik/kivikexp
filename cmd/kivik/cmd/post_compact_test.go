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
	"io/ioutil"
	"net/http"
	"strings"
	"testing"

	"gitlab.com/flimzy/testy"

	"github.com/go-kivik/xkivik/v4/cmd/kivik/errors"
)

func Test_post_compact_RunE(t *testing.T) {
	tests := testy.NewTable()

	tests.Add("missing dsn", cmdTest{
		args:   []string{"post", "compact"},
		status: errors.ErrUsage,
	})
	tests.Add("success", func(t *testing.T) interface{} {
		s := testy.ServeResponseValidator(t, &http.Response{
			Body: ioutil.NopCloser(strings.NewReader(`{"ok":true}`)),
		}, func(t *testing.T, req *http.Request) {
			if req.Method != http.MethodPost {
				t.Errorf("Unexpected method: %v", req.Method)
			}
			if req.URL.Path != "/foo/_compact" {
				t.Errorf("Unexpected path: %s", req.URL.Path)
			}
		})

		return cmdTest{
			args: []string{"post", "compact", s.URL + "/foo"},
		}
	})

	tests.Run(t, func(t *testing.T, tt cmdTest) {
		tt.Test(t)
	})
}
