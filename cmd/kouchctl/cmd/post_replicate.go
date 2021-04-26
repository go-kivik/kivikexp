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

	"github.com/spf13/cobra"

	"github.com/go-kivik/xkivik/v4/cmd/kouchctl/errors"
)

type postReplicate struct {
	*root
	docIDs []string
}

func postReplicateCmd(r *root) *cobra.Command {
	c := &postReplicate{
		root: r,
	}
	cmd := &cobra.Command{
		Use:     "replicate [dsn]",
		Aliases: []string{"rep"},
		Short:   "Replicate a database",
		Long:    "Creates a remotely-managed replication between source and target. `source` and `target` values must be provided via -O flags, and should be URLs or JSON objects.",
		RunE:    c.RunE,
	}

	pf := cmd.PersistentFlags()
	pf.StringSliceVar(&c.docIDs, "doc-id", nil, "Document IDs to be synchronized")

	return cmd
}

func (c *postReplicate) RunE(cmd *cobra.Command, _ []string) error {
	client, err := c.client()
	if err != nil {
		return err
	}

	opts := c.opts()
	source, _ := opts["source"].(string)
	target, _ := opts["target"].(string)
	if source == "" && target == "" {
		return errors.Code(errors.ErrUsage, "explicit source or target required")
	}
	if len(c.docIDs) > 0 {
		opts["doc_ids"] = c.docIDs
	}
	var sObj, tObj map[string]interface{}
	if err := json.Unmarshal([]byte(source), &sObj); err == nil {
		source = ""
		opts["source"] = sObj
	}
	if err := json.Unmarshal([]byte(target), &tObj); err == nil {
		target = ""
		opts["target"] = tObj
	}

	c.log.Debugf("[post] Will replicate %s to %s", source, target)
	return c.retry(func() error {
		_, err := client.Replicate(cmd.Context(), target, source, opts)
		if err != nil {
			return err
		}
		return c.fmt.OK()
	})
}
