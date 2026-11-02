package report

import (
	"bytes"
	"encoding/json"
	"io"
	"testing"

	"configarr/internal/updates"
	"configarr/internal/xmlconfig"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWrite(t *testing.T) {
	t.Parallel()

	t.Run("text hides values by default", func(t *testing.T) {
		t.Parallel()

		var stdout, stderr bytes.Buffer
		results := []updates.Result{{FilePath: "/config.xml", Changes: []xmlconfig.Change{{Path: "/Root/Key", Before: "old-secret", After: "new-secret"}}}}
		err := Write(&stdout, &stderr, results, Text, false)
		require.NoError(t, err)

		assert.Equal(t, "/config.xml: /Root/Key: updated\n", stdout.String())
		assert.Empty(t, stderr.String())
	})

	t.Run("text escapes values and filenames onto one line", func(t *testing.T) {
		t.Parallel()

		var stdout bytes.Buffer
		results := []updates.Result{{FilePath: "config\n.xml", Changes: []xmlconfig.Change{{Path: "/Root/Key", Before: "a\n\"b", After: ""}}}}
		err := Write(&stdout, io.Discard, results, Text, true)
		require.NoError(t, err)

		assert.Equal(t, "config\\n.xml: /Root/Key: \"a\\n\\\"b\" -> \"\"\n", stdout.String())
	})

	t.Run("JSON hides values by default", func(t *testing.T) {
		t.Parallel()

		var stdout bytes.Buffer
		results := []updates.Result{{FilePath: "/config.xml", Changes: []xmlconfig.Change{{Path: "/Root/Key", Before: "old-secret", After: "new-secret"}}}}
		err := Write(&stdout, io.Discard, results, JSON, false)
		require.NoError(t, err)

		assert.JSONEq(t, `{"file":"/config.xml","status":"updated","path":"/Root/Key"}`, stdout.String())
	})

	t.Run("JSON includes empty and multiline values when requested", func(t *testing.T) {
		t.Parallel()

		var stdout bytes.Buffer
		results := []updates.Result{{FilePath: "/config.xml", Changes: []xmlconfig.Change{
			{Path: "/Root/First", Before: "old\nvalue", After: ""},
			{Path: "/Root/Second", Before: "", After: "new"},
		}}}
		err := Write(&stdout, io.Discard, results, JSON, true)
		require.NoError(t, err)

		lines := bytes.Split(bytes.TrimSpace(stdout.Bytes()), []byte("\n"))
		require.Len(t, lines, 2)
		assert.JSONEq(t, `{"file":"/config.xml","status":"updated","path":"/Root/First","before":"old\nvalue","after":""}`, string(lines[0]))
		assert.JSONEq(t, `{"file":"/config.xml","status":"updated","path":"/Root/Second","before":"","after":"new"}`, string(lines[1]))
	})

	t.Run("JSON includes skipped and unchanged files on stdout", func(t *testing.T) {
		t.Parallel()

		var stdout, stderr bytes.Buffer
		results := []updates.Result{{FilePath: "missing.xml", SkippedMissing: true}, {FilePath: "same.xml"}}
		err := Write(&stdout, &stderr, results, JSON, true)
		require.NoError(t, err)

		decoder := json.NewDecoder(&stdout)
		var skipped, unchanged map[string]string
		require.NoError(t, decoder.Decode(&skipped))
		require.NoError(t, decoder.Decode(&unchanged))
		assert.Equal(t, map[string]string{"file": "missing.xml", "status": "skipped"}, skipped)
		assert.Equal(t, map[string]string{"file": "same.xml", "status": "unchanged"}, unchanged)
		assert.Empty(t, stderr.String())
	})

	t.Run("text reports skipped files on stderr", func(t *testing.T) {
		t.Parallel()

		var stdout, stderr bytes.Buffer
		err := Write(&stdout, &stderr, []updates.Result{{FilePath: "missing.xml", SkippedMissing: true}, {FilePath: "same.xml"}}, Text, false)
		require.NoError(t, err)

		assert.Equal(t, "same.xml: no updates made to the configuration.\n", stdout.String())
		assert.Equal(t, "missing.xml: no configuration file found. Skipping update.\n", stderr.String())
	})

	t.Run("returns output errors", func(t *testing.T) {
		t.Parallel()

		err := Write(closedWriter{}, io.Discard, []updates.Result{{FilePath: "config.xml"}}, JSON, false)
		assert.ErrorIs(t, err, io.ErrClosedPipe)
	})
}

type closedWriter struct{}

func (closedWriter) Write([]byte) (int, error) { return 0, io.ErrClosedPipe }
