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

func TestEditFile(t *testing.T) {
	t.Parallel()

	t.Run("File preserves symlink and permissions", func(t *testing.T) {
		t.Parallel()
		directory := t.TempDir()
		target := filepath.Join(directory, "actual.xml")
		link := filepath.Join(directory, "config.xml")
		original := []byte("<?xml version=\"1.0\"?>\n<Settings><!--keep--><Key>old</Key></Settings>\n")
		require.NoError(t, os.WriteFile(target, original, 0o600))
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
	})

	t.Run("Noop preserves file identity", func(t *testing.T) {
		t.Parallel()

		path := filepath.Join(t.TempDir(), "config.xml")
		require.NoError(t, os.WriteFile(path, []byte(`<Anything><Key>old</Key></Anything>`), 0o600))
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
	})

	t.Run("trailing text", func(t *testing.T) {
		t.Parallel()

		source := `<Root><Key>old</Key></Root>garbage`
		path := filepath.Join(t.TempDir(), "config.xml")
		require.NoError(t, os.WriteFile(path, []byte(source), 0o600))

		_, err := EditFile(path, []Update{{ID: "a", Path: "/Root/Key", Value: "new"}})
		require.Error(t, err)

		got, err := os.ReadFile(path)
		require.NoError(t, err)
		assert.Equal(t, source, string(got))
	})

	t.Run("ambiguous target", func(t *testing.T) {
		t.Parallel()

		source := `<Root><Key>old</Key><Key>duplicate</Key></Root>`
		path := filepath.Join(t.TempDir(), "config.xml")
		require.NoError(t, os.WriteFile(path, []byte(source), 0o600))

		_, err := EditFile(path, []Update{{ID: "a", Path: "/Root/Key", Value: "new"}})
		require.Error(t, err)

		got, err := os.ReadFile(path)
		require.NoError(t, err)
		assert.Equal(t, source, string(got))
	})

	t.Run("multiple roots", func(t *testing.T) {
		t.Parallel()

		source := `<Root><Key>old</Key></Root><Other/>`
		path := filepath.Join(t.TempDir(), "config.xml")
		require.NoError(t, os.WriteFile(path, []byte(source), 0o600))

		_, err := EditFile(path, []Update{{ID: "a", Path: "/Root/Key", Value: "new"}})
		require.Error(t, err)

		got, err := os.ReadFile(path)
		require.NoError(t, err)
		assert.Equal(t, source, string(got))
	})

	t.Run("wraps missing file error", func(t *testing.T) {
		t.Parallel()

		path := filepath.Join(t.TempDir(), "missing.xml")

		_, err := EditFile(path, nil)

		assert.ErrorIs(t, err, os.ErrNotExist)
	})

	t.Run("rejects second root without updates", func(t *testing.T) {
		t.Parallel()

		suffix := `<Config/>`
		path := filepath.Join(t.TempDir(), "config.xml")
		source := []byte(`<Config><Key>value</Key></Config>` + suffix)
		require.NoError(t, os.WriteFile(path, source, 0600))

		_, err := EditFile(path, nil)
		assert.Error(t, err)
	})

	t.Run("rejects trailing text without updates", func(t *testing.T) {
		t.Parallel()

		suffix := `garbage`
		path := filepath.Join(t.TempDir(), "config.xml")
		source := []byte(`<Config><Key>value</Key></Config>` + suffix)
		require.NoError(t, os.WriteFile(path, source, 0600))

		_, err := EditFile(path, nil)
		assert.Error(t, err)
	})

	t.Run("rejects malformed element without updates", func(t *testing.T) {
		t.Parallel()

		suffix := `<broken`
		path := filepath.Join(t.TempDir(), "config.xml")
		source := []byte(`<Config><Key>value</Key></Config>` + suffix)
		require.NoError(t, os.WriteFile(path, source, 0600))

		_, err := EditFile(path, nil)
		assert.Error(t, err)
	})
}

func TestCopyEdits(t *testing.T) {
	t.Parallel()

	t.Run("propagates write failure", func(t *testing.T) {
		t.Parallel()

		source := bytes.NewReader([]byte(`<Root>old</Root>`))
		edits, err := planEdits(source, source.Size(), []Update{{ID: "a", Path: "/Root", Value: "new"}})
		require.NoError(t, err)

		err = copyEdits(source, source.Size(), failingWriter{}, edits)

		assert.ErrorIs(t, err, io.ErrClosedPipe)
	})
}

type failingWriter struct{}

func (failingWriter) Write(data []byte) (int, error) { return 0, io.ErrClosedPipe }
