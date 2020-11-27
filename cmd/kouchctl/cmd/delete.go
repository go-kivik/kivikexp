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
)

type delete struct {
	doc *cobra.Command
	db  *cobra.Command
	*root
}

func deleteCmd(r *root) *cobra.Command {
	c := &delete{
		root: r,
		doc:  deleteDocCmd(r),
		db:   deleteDBCmd(r),
	}
	cmd := &cobra.Command{
		Use:     "delete [command]",
		Aliases: []string{"del"},
		Short:   "Delete a resource",
		Long:    `Delete a resource described by the URL`,
		RunE:    c.RunE,
	}

	cmd.AddCommand(c.doc)
	cmd.AddCommand(c.db)

	return cmd
}

func (c *delete) RunE(cmd *cobra.Command, args []string) error {
	if c.conf.HasDoc() {
		return c.doc.RunE(cmd, args)
	}
	if c.conf.HasDB() {
		return c.db.RunE(cmd, args)
	}

	_, err := c.client()
	return err
}
