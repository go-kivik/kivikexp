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
	"bytes"
	"encoding/json"

	"github.com/spf13/cobra"
)

type get struct {
	*root
}

func getCmd(r *root) *cobra.Command {
	g := &get{
		root: r,
	}
	return &cobra.Command{
		Use:   "get [dsn]",
		Short: "get a document",
		Long:  `Fetch a document with the HTTP GET verb`,
		RunE:  g.RunE,
	}
}

func (c *get) RunE(cmd *cobra.Command, _ []string) error {
	db, docID, err := c.conf.DBDoc()
	if err != nil {
		return err
	}
	c.log.Debugf("[get] Will fetch document: %s%s/%s", c.client.DSN(), db, docID)
	return c.retry(func() error {
		row := c.client.DB(db).Get(cmd.Context(), docID)
		if err := row.Err; err != nil {
			return err
		}
		var doc json.RawMessage
		if err := row.ScanDoc(&doc); err != nil {
			return err
		}
		return c.fmt.Output(bytes.NewReader(doc))
	})
}
