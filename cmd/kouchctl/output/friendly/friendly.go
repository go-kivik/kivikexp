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

package friendly

import (
	"io"

	"github.com/go-kivik/xkivik/v4/cmd/kouchctl/output"
	"github.com/go-kivik/xkivik/v4/cmd/kouchctl/output/json"
)

type format struct{}

var _ output.Format = &format{}

// New returns a new friendly formatter.
func New() output.Format {
	return &format{}
}

func (f *format) Output(w io.Writer, r io.Reader) error {
	if f, ok := r.(output.FriendlyOutput); ok {
		return f.Execute(w)
	}
	return json.New().Output(w, r)
}
