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
	"context"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/spf13/cobra"

	"github.com/go-kivik/couchdb/v4"
	"github.com/go-kivik/kivik/v4"

	"github.com/go-kivik/xkivik/v4/cmd/kouchctl/config"
	"github.com/go-kivik/xkivik/v4/cmd/kouchctl/errors"
	"github.com/go-kivik/xkivik/v4/cmd/kouchctl/log"
	"github.com/go-kivik/xkivik/v4/cmd/kouchctl/output"
	"github.com/go-kivik/xkivik/v4/cmd/kouchctl/output/gotmpl"
	"github.com/go-kivik/xkivik/v4/cmd/kouchctl/output/json"
	"github.com/go-kivik/xkivik/v4/cmd/kouchctl/output/raw"
	"github.com/go-kivik/xkivik/v4/cmd/kouchctl/output/yaml"
)

type root struct {
	confFile string
	debug    bool
	log      log.Logger
	conf     *config.Config
	cmd      *cobra.Command
	fmt      *output.Formatter

	client         *kivik.Client
	requestTimeout string
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute(ctx context.Context) {
	fmt.Println(os.Args)
	lg := log.New()
	root := rootCmd(lg)
	os.Exit(root.execute(ctx))
}

func (r *root) execute(ctx context.Context) int {
	err := r.cmd.ExecuteContext(ctx)
	if err == nil {
		return 0
	}
	code := extractExitCode(err)

	return code
}

func extractExitCode(err error) int {
	if code := errors.InspectErrorCode(err); code != 0 {
		return code
	}

	// Any unhandled errors are assumed to be from Cobra, so return a "failed
	// to initialize" error
	return errors.ErrUsage
}

func formatter() *output.Formatter {
	f := output.New()
	f.Register("json", json.New())
	f.Register("raw", raw.New())
	f.Register("yaml", yaml.New())
	f.Register("go-template", gotmpl.New())
	return f
}

func rootCmd(lg log.Logger) *root {
	r := &root{
		log: lg,
		fmt: formatter(),
	}
	r.cmd = &cobra.Command{
		Use:               "kouchctl",
		Short:             "kouchctl facilitates controlling CouchDB instances",
		Long:              `This tool makes it easier to administrate and interact with CouchDB's HTTP API`,
		PersistentPreRunE: r.init,
		RunE:              r.RunE,
	}
	r.conf = config.New(func() {
		r.cmd.SilenceUsage = true
	})

	pf := r.cmd.PersistentFlags()

	r.fmt.ConfigFlags(pf)
	pf.StringVar(&r.confFile, "kouchconfig", "~/.kouchctl/config", "Path to kouchconfig file to use for CLI requests")
	pf.BoolVarP(&r.debug, "debug", "d", false, "Enable debug output")
	pf.StringVar(&r.requestTimeout, "request-timeout", "", "The time limit for each request. May be specified in h, m, s (default), us, ns")

	r.cmd.AddCommand(getCmd(r))
	r.cmd.AddCommand(pingCmd(r))

	return r
}

func parseTimeout(val string) (time.Duration, error) {
	if val == "" {
		return 0, nil
	}
	if d, err := strconv.ParseFloat(val, 64); err == nil {
		if d < 0 {
			return 0, errors.Code(errors.ErrUsage, "negative timeout not permitted")
		}
		return time.Duration(d) * time.Second, nil
	}
	d, err := time.ParseDuration(val)
	if err != nil {
		return 0, errors.Code(errors.ErrUsage, err)
	}
	if d < 0 {
		return 0, errors.Code(errors.ErrUsage, "negative timeout not permitted")
	}
	return d, nil
}

func (r *root) init(cmd *cobra.Command, args []string) error {
	r.log.SetOut(cmd.OutOrStdout())
	r.log.SetErr(cmd.ErrOrStderr())
	r.log.SetDebug(r.debug)

	r.log.Debug("Debug mode enabled")

	requestTimeout, err := parseTimeout(r.requestTimeout)
	if err != nil {
		return err
	}

	if err := r.conf.Read(r.confFile, r.log); err != nil {
		return err
	}

	if len(args) > 0 {
		if err := r.conf.SetURL(args[0]); err != nil {
			return err
		}
	}

	scheme, dsn, err := r.conf.ClientInfo()
	if err != nil {
		return err
	}

	switch scheme {
	case "http", "https", "couch", "couchs", "couchdb", "couchdbs":
		var err error
		r.client, err = kivik.New("couch", dsn, kivik.Options{
			couchdb.OptionHTTPClient: &http.Client{
				Timeout: requestTimeout,
			},
		})
		if err != nil {
			return err
		}
	default:
		return errors.Codef(errors.ErrUsage, "unsupported URL scheme: %s", scheme)
	}

	return nil
}

func (r *root) RunE(cmd *cobra.Command, args []string) error {
	cx, err := r.conf.DSN()
	if err != nil {
		return err
	}
	r.log.Debugf("DSN: %s from %q", cx, r.conf.CurrentContext)

	return nil
}
