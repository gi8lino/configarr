package cli

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseFlags(t *testing.T) {
	setEnvironment(t, nil)

	t.Run("valid flags", func(t *testing.T) {
		args := []string{"--file", "/path/to/config.xml", "--ignore-missing-config"}

		options, err := Parse(args)
		require.NoError(t, err)

		assert.Equal(t, []string{"/path/to/config.xml"}, options.FilePaths)
		assert.True(t, options.IgnoreMissingConfig)
		assert.Empty(t, options.Updates.Paths)
	})

	t.Run("invalid flags", func(t *testing.T) {
		_, err := Parse([]string{"--invalid"})

		assert.Error(t, err)
	})
}

func TestStaticEnvironmentBindings(t *testing.T) {
	setEnvironment(t, []string{"CONFIGARR__FILE=from-env.xml", "CONFIGARR__IGNORE_MISSING_CONFIG=true"})

	options, err := Parse(nil)
	require.NoError(t, err)

	assert.Equal(t, []string{"from-env.xml"}, options.FilePaths)
	assert.True(t, options.IgnoreMissingConfig)

	options, err = Parse([]string{"--file=from-cli.xml"})
	require.NoError(t, err)

	assert.Equal(t, []string{"from-cli.xml"}, options.FilePaths)
}

func TestParseKeepsValueSourceUnresolved(t *testing.T) {
	setEnvironment(t, nil)
	args := []string{"--file=unused.xml", "--update.a.path=/Root/Key", "--update.a.valueFromEnv=UNSET_SOURCE"}

	options, err := Parse(args)
	require.NoError(t, err)

	assert.Equal(t, "UNSET_SOURCE", options.Updates.ValueFromEnv["a"])
}

func TestPrefixFlagRemoved(t *testing.T) {
	setEnvironment(t, nil)

	_, err := Parse([]string{"--file=unused.xml", "--prefix=OLD__"})

	assert.Error(t, err)
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

func TestFileArguments(t *testing.T) {
	tests := []struct {
		name      string
		args      []string
		environ   []string
		want      []string
		wantError bool
	}{
		{name: "one positional file", args: []string{"radarr.xml"}, want: []string{"radarr.xml"}},
		{name: "multiple positional files", args: []string{"radarr.xml", "sonarr.xml"}, want: []string{"radarr.xml", "sonarr.xml"}},
		{name: "repeated file flag", args: []string{"--file=radarr.xml", "--file", "sonarr.xml"}, want: []string{"radarr.xml", "sonarr.xml"}},
		{name: "combined forms", args: []string{"sonarr.xml", "--file=radarr.xml"}, want: []string{"radarr.xml", "sonarr.xml"}},
		{name: "intermixed flags", args: []string{"radarr.xml", "--ignore-missing-config", "sonarr.xml"}, want: []string{"radarr.xml", "sonarr.xml"}},
		{name: "literal filename", args: []string{"--file= radarr, sonarr.xml "}, want: []string{" radarr, sonarr.xml "}},
		{name: "dash filename", args: []string{"--", "-radarr.xml"}, want: []string{"-radarr.xml"}},
		{name: "environment fallback", environ: []string{"CONFIGARR__FILE=radarr.xml"}, want: []string{"radarr.xml"}},
		{name: "flag overrides environment", args: []string{"--file=sonarr.xml"}, environ: []string{"CONFIGARR__FILE=radarr.xml"}, want: []string{"sonarr.xml"}},
		{name: "positional overrides environment", args: []string{"sonarr.xml"}, environ: []string{"CONFIGARR__FILE=radarr.xml"}, want: []string{"sonarr.xml"}},
		{name: "no files", wantError: true},
		{name: "empty flag", args: []string{"--file="}, wantError: true},
		{name: "empty positional file", args: []string{""}, wantError: true},
		{name: "old flag rejected", args: []string{"--config=radarr.xml"}, wantError: true},
		{name: "old environment ignored", environ: []string{"CONFIGARR__CONFIG=radarr.xml"}, wantError: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			setEnvironment(t, test.environ)

			options, err := Parse(test.args)
			if test.wantError {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)

			assert.Equal(t, test.want, options.FilePaths)
		})
	}
}
