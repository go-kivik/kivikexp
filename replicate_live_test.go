// +bui ld livetest

package xkivik

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/flimzy/diff"
	"github.com/flimzy/testy"
	_ "github.com/go-kivik/couchdb" // CouchDB driver
	_ "github.com/go-kivik/fsdb"    // Filesystem driver
	"github.com/go-kivik/kivik"
	"github.com/go-kivik/kiviktest/kt"
)

func TestReplicate_live(t *testing.T) {
	type tt struct {
		source, target *kivik.DB
		options        kivik.Options
		status         int
		err            string
		result         *ReplicationResult
	}
	tests := testy.NewTable()
	tests.Add("couch to couch", func(t *testing.T) interface{} {
		dsn := kt.DSN(t)
		client, err := kivik.New("couch", dsn)
		if err != nil {
			t.Fatal(err)
		}
		sourceName := kt.TestDBName(t)
		targetName := kt.TestDBName(t)
		ctx := context.Background()
		if err := client.CreateDB(ctx, sourceName); err != nil {
			t.Fatal(err)
		}
		tests.Cleanup(func() {
			_ = client.DestroyDB(ctx, sourceName)
		})
		if err := client.CreateDB(ctx, targetName); err != nil {
			t.Fatal(err)
		}
		tests.Cleanup(func() {
			_ = client.DestroyDB(ctx, targetName)
		})
		source := client.DB(ctx, sourceName)
		target := client.DB(ctx, targetName)
		doc := map[string]string{"foo": "bar"}
		if _, err := source.Put(ctx, "foo", doc); err != nil {
			t.Fatal(err)
		}

		return tt{
			source: source,
			target: target,
			result: &ReplicationResult{
				DocsRead:       1,
				DocsWritten:    1,
				MissingChecked: 1,
				MissingFound:   1,
			},
		}
	})
	tests.Add("fs to couch", func(t *testing.T) interface{} {
		fsclient, err := kivik.New("fs", "testdata/")
		if err != nil {
			t.Fatal(err)
		}
		dsn := kt.DSN(t)
		client, err := kivik.New("couch", dsn)
		if err != nil {
			t.Fatal(err)
		}
		ctx := context.Background()
		source := fsclient.DB(ctx, "db1")
		targetName := kt.TestDBName(t)
		if err := client.CreateDB(ctx, targetName); err != nil {
			t.Fatal(err)
		}
		tests.Cleanup(func() {
			_ = client.DestroyDB(ctx, targetName)
		})
		target := client.DB(ctx, targetName)

		return tt{
			source: source,
			target: target,
			result: &ReplicationResult{
				DocsRead:       1,
				DocsWritten:    1,
				MissingChecked: 1,
				MissingFound:   1,
			},
		}
	})
	tests.Add("fs to couch, no shared history", func(t *testing.T) interface{} {
		fsclient, err := kivik.New("fs", "testdata/")
		if err != nil {
			t.Fatal(err)
		}
		dsn := kt.DSN(t)
		client, err := kivik.New("couch", dsn)
		if err != nil {
			t.Fatal(err)
		}
		ctx := context.Background()
		source := fsclient.DB(ctx, "db1")
		targetName := kt.TestDBName(t)
		if err := client.CreateDB(ctx, targetName); err != nil {
			t.Fatal(err)
		}
		tests.Cleanup(func() {
			_ = client.DestroyDB(ctx, targetName)
		})
		target := client.DB(ctx, targetName)

		if _, err := Replicate(ctx, target, source); err != nil {
			t.Fatalf("setup replication failed: %s", err)
		}

		return tt{
			source: fsclient.DB(ctx, "db2"),
			target: target,
			result: &ReplicationResult{
				DocsRead:       1,
				DocsWritten:    1,
				MissingChecked: 1,
				MissingFound:   1,
			},
		}
	})
	tests.Add("couch to couch with sec", func(t *testing.T) interface{} {
		dsn := kt.DSN(t)
		client, err := kivik.New("couch", dsn)
		if err != nil {
			t.Fatal(err)
		}
		sourceName := kt.TestDBName(t)
		targetName := kt.TestDBName(t)
		ctx := context.Background()
		if err := client.CreateDB(ctx, sourceName); err != nil {
			t.Fatal(err)
		}
		tests.Cleanup(func() {
			_ = client.DestroyDB(ctx, sourceName)
		})
		if err := client.CreateDB(ctx, targetName); err != nil {
			t.Fatal(err)
		}
		tests.Cleanup(func() {
			_ = client.DestroyDB(ctx, targetName)
		})
		source := client.DB(ctx, sourceName)
		target := client.DB(ctx, targetName)
		doc := map[string]string{"foo": "bar"}
		if _, err := source.Put(ctx, "foo", doc); err != nil {
			t.Fatal(err)
		}
		err = source.SetSecurity(ctx, &kivik.Security{
			Members: kivik.Members{
				Names: []string{"bob"},
			},
		})
		if err != nil {
			t.Fatal(err)
		}

		return tt{
			source:  source,
			target:  target,
			options: map[string]interface{}{"copy_security": true},
			result: &ReplicationResult{
				DocsRead:       1,
				DocsWritten:    1,
				MissingChecked: 1,
				MissingFound:   1,
			},
		}
	})
	tests.Add("fs to couch, bad stub", func(t *testing.T) interface{} {
		fsclient, err := kivik.New("fs", "testdata/")
		if err != nil {
			t.Fatal(err)
		}
		dsn := kt.DSN(t)
		client, err := kivik.New("couch", dsn)
		if err != nil {
			t.Fatal(err)
		}
		ctx := context.Background()
		targetName := kt.TestDBName(t)
		if err := client.CreateDB(ctx, targetName); err != nil {
			t.Fatal(err)
		}
		tests.Cleanup(func() {
			_ = client.DestroyDB(ctx, targetName)
		})
		target := client.DB(ctx, targetName)

		return tt{
			source: fsclient.DB(ctx, "db3"),
			target: target,
			result: &ReplicationResult{
				DocsRead:       1,
				DocsWritten:    1,
				MissingChecked: 1,
				MissingFound:   1,
			},
			status: http.StatusBadRequest,
			err:    "Bad Request: Bad special document member: _invalid",
		}
	})
	tests.Add("fs to couch with attachment", func(t *testing.T) interface{} {
		fsclient, err := kivik.New("fs", "testdata/")
		if err != nil {
			t.Fatal(err)
		}
		dsn := kt.DSN(t)
		client, err := kivik.New("couch", dsn)
		if err != nil {
			t.Fatal(err)
		}
		ctx := context.Background()
		source := fsclient.DB(ctx, "db4")
		targetName := kt.TestDBName(t)
		if err := client.CreateDB(ctx, targetName); err != nil {
			t.Fatal(err)
		}
		tests.Cleanup(func() {
			_ = client.DestroyDB(ctx, targetName)
		})
		target := client.DB(ctx, targetName)

		return tt{
			source: source,
			target: target,
			result: &ReplicationResult{
				DocsRead:       1,
				DocsWritten:    1,
				MissingChecked: 1,
				MissingFound:   1,
			},
		}
	})

	tests.Run(t, func(t *testing.T, tt tt) {
		ctx := context.TODO()
		result, err := Replicate(ctx, tt.target, tt.source, tt.options)
		testy.StatusError(t, tt.err, tt.status, err)

		verifyDoc(ctx, t, tt.target, tt.source, "foo")
		verifySec(ctx, t, tt.target)
		result.StartTime = time.Time{}
		result.EndTime = time.Time{}
		if d := diff.AsJSON(tt.result, result); d != nil {
			t.Error(d)
		}
	})
}

func verifyDoc(ctx context.Context, t *testing.T, target, source *kivik.DB, docID string) {
	t.Helper()
	var targetDoc, sourceDoc interface{}
	if err := source.Get(ctx, docID).ScanDoc(&sourceDoc); err != nil {
		t.Fatalf("get %s from source failed: %s", docID, err)
	}
	if err := target.Get(ctx, docID).ScanDoc(&targetDoc); err != nil {
		t.Fatalf("get %s from target failed: %s", docID, err)
	}
	if d := diff.AsJSON(sourceDoc, targetDoc); d != nil {
		t.Error(d)
	}
}

func verifySec(ctx context.Context, t *testing.T, target *kivik.DB) {
	sec, err := target.Security(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if d := diff.AsJSON(&diff.File{Path: "testdata/" + testy.Stub(t) + ".security"}, sec); d != nil {
		t.Errorf("Security object:\n%s", d)
	}
}
