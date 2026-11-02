package app

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"configarr/internal/testutil"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRun(t *testing.T) {
	t.Parallel()

	t.Run("reports completed changes as JSON before a later file failure", func(t *testing.T) {
		t.Parallel()

		if !testutil.WithEnvironment(t, nil) {
			return
		}
		directory := t.TempDir()
		first := filepath.Join(directory, "first.xml")
		missing := filepath.Join(directory, "missing.xml")
		require.NoError(t, os.WriteFile(first, []byte(`<Root><Key>old</Key></Root>`), 0o600))

		var stdout, stderr bytes.Buffer
		err := Run([]string{first, missing, "--output=json", "--show-values", "--update.a.path=/Root/Key", "--update.a.value=new"}, &stdout, &stderr)
		require.ErrorIs(t, err, os.ErrNotExist)

		var entry map[string]string
		require.NoError(t, json.Unmarshal(stdout.Bytes(), &entry))
		assert.Equal(t, map[string]string{
			"file": first, "status": "updated", "path": "/Root/Key", "before": "old", "after": "new",
		}, entry)
		assert.Contains(t, stderr.String(), missing)
		content, err := os.ReadFile(first)
		require.NoError(t, err)
		assert.Equal(t, `<Root><Key>new</Key></Root>`, string(content))
	})

	t.Run("updates XML and preserves permissions", func(t *testing.T) {
		t.Parallel()

		if !testutil.WithEnvironment(t, []string{"SECRET=new<&"}) {
			return
		}
		path := filepath.Join(t.TempDir(), "config.xml")
		original := []byte("<?xml version=\"1.0\"?>\n<Root><!--keep--><Key>old</Key><Other/></Root>\n")
		require.NoError(t, os.WriteFile(path, original, 0600))

		var output bytes.Buffer
		args := []string{"--file", path, "--update.a.path=/Root/Key", "--update.a.valueFromEnv=SECRET"}
		err := Run(args, &output, &output)
		require.NoError(t, err)

		got, err := os.ReadFile(path)
		require.NoError(t, err)
		want := bytes.Replace(original, []byte("old"), []byte("new&lt;&amp;"), 1)
		assert.Equal(t, want, got)
		assert.NotContains(t, output.String(), "new<&", "secret leaked")

		info, err := os.Stat(path)
		require.NoError(t, err)
		assert.Equal(t, os.FileMode(0600), info.Mode().Perm())
	})

	t.Run("failed update leaves original content untouched", func(t *testing.T) {
		t.Parallel()

		if !testutil.WithEnvironment(t, []string{"SECRET=another-secret"}) {
			return
		}

		path := filepath.Join(t.TempDir(), "config.xml")
		original := []byte(`<Root><Key>old</Key></Root>`)
		require.NoError(t, os.WriteFile(path, original, 0600))

		var output bytes.Buffer
		args := []string{
			"--file", path,
			"--update.a.path=/Root/Key", "--update.a.valueFromEnv=SECRET",
			"--update.z.path=/Root/Missing", "--update.z.value=bad",
		}
		err := Run(args, &output, &output)
		require.Error(t, err)

		got, err := os.ReadFile(path)
		require.NoError(t, err)
		assert.Equal(t, original, got)
		assert.NotContains(t, output.String(), "another-secret", "secret leaked")
	})

	t.Run("prints dynamic update help", func(t *testing.T) {
		t.Parallel()

		if !testutil.WithEnvironment(t, nil) {
			return
		}
		var output bytes.Buffer

		err := Run([]string{"--help"}, &output, &output)
		require.NoError(t, err)

		assert.Contains(t, output.String(), "path")
		assert.Contains(t, output.String(), "value")
		assert.Contains(t, output.String(), "valueFromEnv")
	})

	t.Run("rejects missing arguments", func(t *testing.T) {
		t.Parallel()

		if !testutil.WithEnvironment(t, nil) {
			return
		}
		err := Run(nil, &bytes.Buffer{}, &bytes.Buffer{})

		assert.Error(t, err)
	})

	t.Run("preserves document without updates", func(t *testing.T) {
		t.Parallel()

		if !testutil.WithEnvironment(t, nil) {
			return
		}
		path := filepath.Join(t.TempDir(), "config.xml")
		source := []byte(`<?xml version="1.0"?><p:Settings xmlns:p="urn:test"><!--keep--><p:Database host="db"><p:Key>value</p:Key></p:Database></p:Settings>`)
		require.NoError(t, os.WriteFile(path, source, 0600))

		err := Run([]string{"--file", path}, &bytes.Buffer{}, &bytes.Buffer{})
		require.NoError(t, err)

		got, err := os.ReadFile(path)
		require.NoError(t, err)
		assert.Equal(t, source, got)
	})

	t.Run("returns and reports file errors", func(t *testing.T) {
		t.Parallel()

		if !testutil.WithEnvironment(t, nil) {
			return
		}
		var stdout, stderr bytes.Buffer
		args := []string{"--file", filepath.Join(t.TempDir(), "missing.xml")}

		err := Run(args, &stdout, &stderr)

		require.ErrorIs(t, err, os.ErrNotExist)
		assert.Empty(t, stdout.String())
		assert.Contains(t, stderr.String(), err.Error())
	})

	t.Run("ignores old environment format", func(t *testing.T) {
		t.Parallel()

		if !testutil.WithEnvironment(t, []string{"CONFIGARR__A=Key=old-format"}) {
			return
		}
		path := filepath.Join(t.TempDir(), "config.xml")
		source := []byte(`<Root><Key>original</Key></Root>`)
		require.NoError(t, os.WriteFile(path, source, 0600))

		err := Run([]string{"--file", path}, &bytes.Buffer{}, &bytes.Buffer{})
		require.NoError(t, err)

		got, err := os.ReadFile(path)
		require.NoError(t, err)
		assert.Equal(t, source, got)
	})

	t.Run("updates multiple files", func(t *testing.T) {
		t.Parallel()

		if !testutil.WithEnvironment(t, nil) {
			return
		}
		directory := t.TempDir()
		radarr := filepath.Join(directory, "radarr.xml")
		sonarr := filepath.Join(directory, "sonarr.xml")
		source := []byte(`<Config><LaunchBrowser>True</LaunchBrowser></Config>`)
		require.NoError(t, os.WriteFile(radarr, source, 0600))
		require.NoError(t, os.WriteFile(sonarr, source, 0600))

		var stdout, stderr bytes.Buffer
		args := []string{radarr, sonarr, "--update.browser.path=/Config/LaunchBrowser", "--update.browser.value=False"}
		err := Run(args, &stdout, &stderr)
		require.NoError(t, err)

		radarrContent, err := os.ReadFile(radarr)
		require.NoError(t, err)
		assert.Equal(t, `<Config><LaunchBrowser>False</LaunchBrowser></Config>`, string(radarrContent))
		assert.Contains(t, stdout.String(), radarr)

		sonarrContent, err := os.ReadFile(sonarr)
		require.NoError(t, err)
		assert.Equal(t, `<Config><LaunchBrowser>False</LaunchBrowser></Config>`, string(sonarrContent))
		assert.Contains(t, stdout.String(), sonarr)
		assert.Empty(t, stderr.String())
	})

	t.Run("skips missing file and continues", func(t *testing.T) {
		t.Parallel()

		if !testutil.WithEnvironment(t, nil) {
			return
		}
		directory := t.TempDir()
		missing := filepath.Join(directory, "missing.xml")
		existing := filepath.Join(directory, "existing.xml")
		require.NoError(t, os.WriteFile(existing, []byte(`<Root><Key>old</Key></Root>`), 0600))

		var stdout, stderr bytes.Buffer
		args := []string{"--file=" + missing, "--file=" + existing, "--ignore-missing-config", "--update.key.path=/Root/Key", "--update.key.value=new"}
		err := Run(args, &stdout, &stderr)
		require.NoError(t, err)

		content, err := os.ReadFile(existing)
		require.NoError(t, err)
		assert.Equal(t, `<Root><Key>new</Key></Root>`, string(content))
		assert.Contains(t, stderr.String(), missing)
		assert.Contains(t, stdout.String(), existing)
		assert.NoFileExists(t, missing)
	})

	t.Run("stops on file failure", func(t *testing.T) {
		t.Parallel()

		if !testutil.WithEnvironment(t, nil) {
			return
		}
		directory := t.TempDir()
		missing := filepath.Join(directory, "missing.xml")
		existing := filepath.Join(directory, "existing.xml")
		source := []byte(`<Root><Key>old</Key></Root>`)
		require.NoError(t, os.WriteFile(existing, source, 0600))

		var stdout, stderr bytes.Buffer
		args := []string{missing, existing, "--update.key.path=/Root/Key", "--update.key.value=new"}
		err := Run(args, &stdout, &stderr)
		require.ErrorIs(t, err, os.ErrNotExist)

		content, err := os.ReadFile(existing)
		require.NoError(t, err)
		assert.Equal(t, source, content)
		assert.Contains(t, stderr.String(), missing)
		assert.Empty(t, stdout.String())
	})
}
