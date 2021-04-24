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
	"github.com/spf13/cobra"

	"github.com/go-kivik/xkivik/v4/cmd/kouchctl/output"
)

type getSec struct {
	*root
}

func getSecurityCmd(r *root) *cobra.Command {
	g := &getSec{
		root: r,
	}
	return &cobra.Command{
		Use:     "security [dsn]/[database]",
		Aliases: []string{"sec"},
		Short:   "Get a database's security object",
		RunE:    g.RunE,
	}
}

func (c *getSec) RunE(cmd *cobra.Command, _ []string) error {
	client, err := c.client()
	if err != nil {
		return err
	}
	db, err := c.conf.DB()
	if err != nil {
		return err
	}
	c.log.Debugf("[get] Will fetch security object: %s/%s", client.DSN(), db)
	return c.retry(func() error {
		sec, err := client.DB(db).Security(cmd.Context())
		if err != nil {
			return err
		}
		return c.fmt.Output(output.JSONReader(sec))
	})
}
