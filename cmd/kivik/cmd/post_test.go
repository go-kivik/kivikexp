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
	"net/http/httptest"
	"strings"
	"testing"

	"gitlab.com/flimzy/testy"

	"github.com/go-kivik/xkivik/v4/cmd/kivik/errors"
)

func Test_post_RunE(t *testing.T) {
	tests := testy.NewTable()

	tests.Add("missing resource", cmdTest{
		args:   []string{"post"},
		status: errors.ErrUsage,
	})
	tests.Add("auto create doc", func(t *testing.T) interface{} {
		s := testy.ServeResponseValidator(t, &http.Response{
			Body: ioutil.NopCloser(strings.NewReader(`{"ok":true,"id":"random","rev":"1-xxx"}`)),
		}, func(t *testing.T, req *http.Request) {
			defer req.Body.Close() // nolint:errcheck
			if d := testy.DiffAsJSON(testy.Snapshot(t), req.Body); d != nil {
				t.Error(d)
			}
		})

		return cmdTest{
			args: []string{"--debug", "post", s.URL + "/foo", "--data", `{"foo":"bar"}`},
		}
	})
	tests.Add("auto view cleanup", func(t *testing.T) interface{} {
		s := testy.ServeResponseValidator(t, &http.Response{
			Body: ioutil.NopCloser(strings.NewReader(`{"ok":true,"id":"random","rev":"1-xxx"}`)),
		}, func(t *testing.T, req *http.Request) {
			if req.Method != http.MethodPost {
				t.Errorf("Unexpected method: %s", req.Method)
			}
		})

		return cmdTest{
			args: []string{"post", s.URL + "/foo/_view_cleanup"},
		}
	})
	tests.Add("auto flush", func(t *testing.T) interface{} {
		s := testy.ServeResponseValidator(t, &http.Response{
			Body: ioutil.NopCloser(strings.NewReader(`{"ok":true}`)),
		}, func(t *testing.T, req *http.Request) {
			if req.Method != http.MethodPost {
				t.Errorf("Unexpected method: %v", req.Method)
			}
			if req.URL.Path != "/foo/_ensure_full_commit" {
				t.Errorf("Unexpected path: %s", req.URL.Path)
			}
		})

		return cmdTest{
			args: []string{"post", s.URL + "/foo/_ensure_full_commit"},
		}
	})
	tests.Add("auto compact", func(t *testing.T) interface{} {
		s := testy.ServeResponseValidator(t, &http.Response{
			Body: ioutil.NopCloser(strings.NewReader(`{"ok":true}`)),
		}, func(t *testing.T, req *http.Request) {
			if req.Method != http.MethodPost {
				t.Errorf("Unexpected method: %v", req.Method)
			}
			if req.URL.Path != "/asdf/_compact" {
				t.Errorf("Unexpected path: %s", req.URL.Path)
			}
		})

		return cmdTest{
			args: []string{"post", s.URL + "/asdf/_compact"},
		}
	})
	tests.Add("auto compact views", func(t *testing.T) interface{} {
		s := testy.ServeResponseValidator(t, &http.Response{
			Body: ioutil.NopCloser(strings.NewReader(`{"ok":true}`)),
		}, func(t *testing.T, req *http.Request) {
			if req.Method != http.MethodPost {
				t.Errorf("Unexpected method: %v", req.Method)
			}
			if req.URL.Path != "/asdf/_compact/foo" {
				t.Errorf("Unexpected path: %s", req.URL.Path)
			}
		})

		return cmdTest{
			args: []string{"post", s.URL + "/asdf/_compact/foo"},
		}
	})
	tests.Add("auto purge", func(t *testing.T) interface{} {
		s := testy.ServeResponseValidator(t, &http.Response{
			Body: ioutil.NopCloser(strings.NewReader(`{"ok":true}`)),
		}, func(t *testing.T, req *http.Request) {
			if req.Method != http.MethodPost {
				t.Errorf("Unexpected method: %v", req.Method)
			}
			if req.URL.Path != "/db/_purge" {
				t.Errorf("Unexpected path: %s", req.URL.Path)
			}
			if d := testy.DiffAsJSON(testy.Snapshot(t), req.Body); d != nil {
				t.Error(d)
			}
		})

		return cmdTest{
			args: []string{"post", s.URL + "/db/_purge", "--data", `{"foo":["1-xxx"]}`},
		}
	})
	tests.Add("auto replicate", func(t *testing.T) interface{} {
		s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodHead {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			if r.Method != http.MethodPost {
				t.Errorf("Unexpected method: %s", r.Method)
			}
			defer r.Body.Close() // nolint:errcheck
			if d := testy.DiffAsJSON(testy.Snapshot(t), r.Body); d != nil {
				t.Error(d)
			}
			_, _ = w.Write([]byte(`{"ok":true,"session_id":"87bf1c2a565f20976c4cb19a22528b7e","source_last_seq":"6-g1AAAABteJzLYWBgYMpgTmHgzcvPy09JdcjLz8gvLskBCScyJNX___8_K4M5kS0XKMBunmRiYmRmhK4Yh_Y8FiDJ0ACk_oNMSWTIAgDY6SGt","replication_id_version":4,"history":[{"session_id":"87bf1c2a565f20976c4cb19a22528b7e","start_time":"Sun, 25 Apr 2021 19:53:34 GMT","end_time":"Sun, 25 Apr 2021 19:53:35 GMT","start_last_seq":0,"end_last_seq":"6-g1AAAABteJzLYWBgYMpgTmHgzcvPy09JdcjLz8gvLskBCScyJNX___8_K4M5kS0XKMBunmRiYmRmhK4Yh_Y8FiDJ0ACk_oNMSWTIAgDY6SGt","recorded_seq":"6-g1AAAABteJzLYWBgYMpgTmHgzcvPy09JdcjLz8gvLskBCScyJNX___8_K4M5kS0XKMBunmRiYmRmhK4Yh_Y8FiDJ0ACk_oNMSWTIAgDY6SGt","missing_checked":2,"missing_found":2,"docs_read":2,"docs_written":2,"doc_write_failures":0}]}
			`))
		}))

		return cmdTest{
			args: []string{"--debug", "post", s.URL + "/_replicate", "-O", "source=http://example.com/foo", "-O", "target=http://example.com/bar"},
		}
	})
	tests.Run(t, func(t *testing.T, tt cmdTest) {
		tt.Test(t)
	})
}
