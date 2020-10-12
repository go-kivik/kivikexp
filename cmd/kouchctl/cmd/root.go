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
	"math/rand"
	"net"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/cenkalti/backoff/v4"
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

	requestTimeout string
	retryDelay     string
	connectTimeout string
	retryTimeout   string

	client *kivik.Client

	// retry attempts
	retryCount         int
	retryDelayParsed   time.Duration
	retryTimeoutParsed time.Duration
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute(ctx context.Context) {
	// Init the pRNG here, so it doesn't affect tests.
	rand.Seed(time.Now().Unix())
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
	pf.IntVar(&r.retryCount, "retry", 0, "In case of transient error, retry up to this many times. A negative value retries forever.")

	// Timeouts
	// Might consider adding:
	// - http.Transport.TLSHandshakeTimeout
	// - http.Transport.ResponseHeaderTimeout
	// - http.Transport.ExpectContinueTimeout (not sure this is relevant, as I'm not sure CouchDB ever uses a 100)
	// - Read timeout (would have to be an HTTP transport that wraps the Response.Body reader with a context-aware reader that extends the timeout every time more data is read)
	pf.StringVar(&r.requestTimeout, "request-timeout", "", "The time limit for each request.")
	pf.StringVar(&r.retryDelay, "retry-delay", "", "Delay between retry attempts. Disables the default exponential backoff algorithm.")
	pf.StringVar(&r.connectTimeout, "connect-timeout", "", "Limits the time spent establishing a TCP connection.")
	pf.StringVar(&r.retryTimeout, "retry-timeout", "", "When used with --retry, no more retries will be attempted after this timeout.")

	r.cmd.AddCommand(getCmd(r))
	r.cmd.AddCommand(pingCmd(r))

	return r
}

func parseDuration(val string) (time.Duration, error) {
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

	requestTimeout, err := parseDuration(r.requestTimeout)
	if err != nil {
		return err
	}
	connectTimeout, err := parseDuration(r.connectTimeout)
	if err != nil {
		return err
	}
	r.retryDelayParsed, err = parseDuration(r.retryDelay)
	if err != nil {
		return err
	}
	r.retryTimeoutParsed, err = parseDuration(r.retryTimeout)
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
				Transport: &http.Transport{
					DialContext: (&net.Dialer{
						Timeout: connectTimeout,
					}).DialContext,
				},
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

func (r *root) retry(fn func() error) error {
	if r.retryCount == 0 {
		return fn()
	}
	var bo backoff.BackOff
	switch {
	case r.retryDelayParsed == 0 && r.retryDelay != "": // Disables retry delay
		bo = &backoff.ZeroBackOff{}
	case r.retryDelayParsed != 0:
		bo = backoff.NewConstantBackOff(r.retryDelayParsed)
	default:
		bo = backoff.NewExponentialBackOff()
	}
	if r.retryCount >= 0 {
		// WithMaxRetries really means WithMaxTries, so +1
		bo = backoff.WithMaxRetries(bo, uint64(r.retryCount+1))
	}
	if r.retryTimeoutParsed > 0 {
		ctx := context.Background()
		ctx, cancel := context.WithTimeout(ctx, r.retryTimeoutParsed)
		defer cancel()
		bo = backoff.WithContext(bo, ctx)
	}
	var count int
	var err error
	return backoff.Retry(func() error {
		if count > 0 {
			msg := fmt.Sprintf("Warning: Transient problem: %s.", err)
			switch nbo := bo.NextBackOff(); nbo {
			case backoff.Stop, 0:
			default:
				msg += fmt.Sprintf(" Will retry in %s.", fmtDuration(nbo))
			}
			if remain := r.retryCount - count; remain > 0 {
				msg += fmt.Sprintf(" %d retries left.", remain)
			}
			r.log.Info(msg)
		}
		count++
		fmt.Println(count)
		err = fn()
		return err
	}, bo)
}

// nolint:gomnd
func fmtDuration(dur time.Duration) string {
	s := dur.Seconds()
	if s < 60 {
		return fmt.Sprintf("%0.2fs", s)
	}
	m := int(s / 60)
	s -= float64(m) * 60
	if m < 60 {
		return fmt.Sprintf("%dm%ds", m, int(s))
	}
	h := m / 60
	m -= h * 60
	if h < 24 {
		return fmt.Sprintf("%dh%dm", h, m)
	}
	d := h / 24
	h -= d * 24
	return fmt.Sprintf("%dd%dh%dm", d, h, m)
}
