package xmlconfig

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEditFilePreservesSymlinkAndPermissions(t *testing.T) {
	directory := t.TempDir()
	target := filepath.Join(directory, "actual.xml")
	link := filepath.Join(directory, "config.xml")
	original := []byte("<?xml version=\"1.0\"?>\n<Settings><!--keep--><Key>old</Key></Settings>\n")
	require.NoError(t, os.WriteFile(target, original, 0600))
	require.NoError(t, os.Symlink("actual.xml", link))
	before, err := os.Stat(target)
	require.NoError(t, err)

	changed, err := EditFile(link, []Update{{ID: "a", Path: "/Settings/Key", Value: "new&value"}})
	require.NoError(t, err)
	assert.True(t, changed)

	got, err := os.ReadFile(target)
	require.NoError(t, err)
	want := bytes.Replace(original, []byte("old"), []byte("new&amp;value"), 1)
	assert.Equal(t, want, got, "unrelated bytes must be preserved")

	after, err := os.Stat(target)
	require.NoError(t, err)
	assert.Equal(t, before.Mode(), after.Mode())
	value, err := os.Readlink(link)
	require.NoError(t, err)
	assert.Equal(t, "actual.xml", value)

	temporary, err := filepath.Glob(filepath.Join(directory, ".configarr-*"))
	require.NoError(t, err)
	assert.Empty(t, temporary, "temporary files must be removed")
}

func TestFinalNoopPreservesFileIdentity(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.xml")
	require.NoError(t, os.WriteFile(path, []byte(`<Anything><Key>old</Key></Anything>`), 0600))
	before, err := os.Stat(path)
	require.NoError(t, err)

	changed, err := EditFile(path, []Update{
		{ID: "a", Path: "/Anything/Key", Value: "new"},
		{ID: "z", Path: "/Anything/Key", Value: "old"},
	})
	require.NoError(t, err)
	assert.False(t, changed)

	after, err := os.Stat(path)
	require.NoError(t, err)
	assert.True(t, os.SameFile(before, after), "no-op must keep the original inode")
	assert.Equal(t, before.ModTime(), after.ModTime())
}

func TestOriginalDocumentSelectors(t *testing.T) {
	source := []byte(`<Root><Key name="old">old</Key></Root>`)
	reader := bytes.NewReader(source)
	updates := []Update{
		{ID: "rename", Path: "/Root/Key/@name", Value: "new"},
		{ID: "text", Path: "/Root/Key[@name='old']", Value: "new"},
	}

	edits, err := planEdits(reader, int64(len(source)), updates)
	require.NoError(t, err)
	var result bytes.Buffer
	require.NoError(t, copyEdits(reader, int64(len(source)), &result, edits))

	assert.Equal(t, `<Root><Key name="new">new</Key></Root>`, result.String())
}

func TestInvalidDocumentDoesNotWrite(t *testing.T) {
	documents := []string{
		`<Root><Key>old</Key></Root>garbage`,
		`<Root><Key>old</Key><Key>duplicate</Key></Root>`,
		`<Root><Key>old</Key></Root><Other/>`,
	}

	for _, source := range documents {
		t.Run(source, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "config.xml")
			require.NoError(t, os.WriteFile(path, []byte(source), 0600))

			_, err := EditFile(path, []Update{{ID: "a", Path: "/Root/Key", Value: "new"}})
			require.Error(t, err)

			got, err := os.ReadFile(path)
			require.NoError(t, err)
			assert.Equal(t, source, string(got))
		})
	}
}

func TestMissingFileErrorIsWrapped(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing.xml")

	_, err := EditFile(path, nil)

	assert.ErrorIs(t, err, os.ErrNotExist)
}

func TestCopyEditsPropagatesWriteFailure(t *testing.T) {
	source := bytes.NewReader([]byte(`<Root>old</Root>`))
	edits, err := planEdits(source, source.Size(), []Update{{ID: "a", Path: "/Root", Value: "new"}})
	require.NoError(t, err)

	err = copyEdits(source, source.Size(), failingWriter{}, edits)

	assert.ErrorIs(t, err, io.ErrClosedPipe)
}

type failingWriter struct{}

func (failingWriter) Write(data []byte) (int, error) { return 0, io.ErrClosedPipe }
