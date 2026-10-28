package app

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDynamicRun(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.xml")
	original := []byte("<?xml version=\"1.0\"?>\n<Root><!--keep--><Key>old</Key><Other/></Root>\n")
	require.NoError(t, os.WriteFile(path, original, 0600))

	var output bytes.Buffer
	args := []string{"configarr", "--file", path, "--update.a.path=/Root/Key", "--update.a.valueFromEnv=SECRET"}
	err := runWithEnvironment(t, []string{"SECRET=new<&"}, args, &output)
	require.NoError(t, err)

	got, err := os.ReadFile(path)
	require.NoError(t, err)
	want := bytes.Replace(original, []byte("old"), []byte("new&lt;&amp;"), 1)
	assert.Equal(t, want, got)
	assert.NotContains(t, output.String(), "new<&", "secret leaked")

	info, err := os.Stat(path)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0600), info.Mode().Perm())

	// A failed later update must leave the previously saved content untouched.
	before := bytes.Clone(got)
	args = append(args, "--update.z.path=/Root/Missing", "--update.z.value=bad")
	err = runWithEnvironment(t, []string{"SECRET=another-secret"}, args, &output)
	require.Error(t, err)

	got, err = os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, before, got)
}

func TestDynamicHelp(t *testing.T) {
	var output bytes.Buffer

	err := runWithEnvironment(t, nil, []string{"configarr", "--help"}, &output)
	require.NoError(t, err)

	for _, field := range []string{"path", "value", "valueFromEnv"} {
		assert.Contains(t, output.String(), field)
	}
}

func TestRunWithoutArguments(t *testing.T) {
	err := runWithEnvironment(t, nil, nil, &bytes.Buffer{})

	assert.Error(t, err)
}

func TestGenericDocumentWithoutUpdates(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.xml")
	source := []byte(`<?xml version="1.0"?><p:Settings xmlns:p="urn:test"><!--keep--><p:Database host="db"><p:Key>value</p:Key></p:Database></p:Settings>`)
	require.NoError(t, os.WriteFile(path, source, 0600))

	err := runWithEnvironment(t, nil, []string{"configarr", "--file", path}, &bytes.Buffer{})
	require.NoError(t, err)

	got, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, source, got)
}

func TestFileErrorsAreReturnedAndReported(t *testing.T) {
	setEnvironment(t, nil)
	var stdout, stderr bytes.Buffer
	args := []string{"--file", filepath.Join(t.TempDir(), "missing.xml")}

	err := Run(args, &stdout, &stderr)

	require.ErrorIs(t, err, os.ErrNotExist)
	assert.Empty(t, stdout.String())
	assert.Contains(t, stderr.String(), err.Error())
}

func TestOldEnvironmentFormatHasNoEffect(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.xml")
	source := []byte(`<Root><Key>original</Key></Root>`)
	require.NoError(t, os.WriteFile(path, source, 0600))

	err := runWithEnvironment(t, []string{"CONFIGARR__A=Key=old-format"}, []string{"configarr", "--file", path}, &bytes.Buffer{})
	require.NoError(t, err)

	got, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, source, got)
}

func setEnvironment(t *testing.T, entries []string) {
	t.Helper()

	keys := map[string]bool{"TOKEN": true, "SECRET": true, "DATABASE_HOST": true}
	for _, entry := range os.Environ() {
		key, _, _ := strings.Cut(entry, "=")
		if strings.HasPrefix(key, "CONFIGARR_") {
			keys[key] = true
		}
	}
	for _, entry := range entries {
		key, _, _ := strings.Cut(entry, "=")
		keys[key] = true
	}

	for key := range keys {
		t.Setenv(key, "")
		require.NoError(t, os.Unsetenv(key))
	}
	for _, entry := range entries {
		key, value, valid := strings.Cut(entry, "=")
		if valid {
			t.Setenv(key, value)
		}
	}
}

func runWithEnvironment(t *testing.T, environ, args []string, output io.Writer) error {
	t.Helper()
	setEnvironment(t, environ)

	if len(args) > 0 {
		args = args[1:]
	}
	return Run(args, output, output)
}

func TestRunUpdatesMultipleFiles(t *testing.T) {
	setEnvironment(t, nil)
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

	for _, file := range []string{radarr, sonarr} {
		content, err := os.ReadFile(file)
		require.NoError(t, err)
		assert.Equal(t, `<Config><LaunchBrowser>False</LaunchBrowser></Config>`, string(content))
		assert.Contains(t, stdout.String(), file)
	}
	assert.Empty(t, stderr.String())
}

func TestRunSkipsMissingFileAndContinues(t *testing.T) {
	setEnvironment(t, nil)
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
}

func TestRunStopsOnFileFailure(t *testing.T) {
	setEnvironment(t, nil)
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
}
