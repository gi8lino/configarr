package updates_test

import (
	"os"
	"strings"
	"testing"

	"configarr/internal/cli"
	"configarr/internal/updates"
	"configarr/internal/xmlconfig"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDynamicValueSources(t *testing.T) {
	tests := []struct {
		name      string
		args      []string
		environ   []string
		want      string
		wantError bool
	}{
		{name: "literal", args: []string{"--update.a.path=/Root/Key", "--update.a.value=hello=world"}, want: "hello=world"},
		{name: "empty literal", args: []string{"--update.a.path=/Root/Key", "--update.a.value="}},
		{name: "environment source", args: []string{"--update.a.path=/Root/Key", "--update.a.valueFromEnv=TOKEN"}, environ: []string{"TOKEN= secret=&value "}, want: " secret=&value "},
		{name: "empty environment source", args: []string{"--update.a.path=/Root/Key", "--update.a.valueFromEnv=TOKEN"}, environ: []string{"TOKEN="}},
		{name: "unset environment source", args: []string{"--update.a.path=/Root/Key", "--update.a.valueFromEnv=TOKEN"}, wantError: true},
		{name: "empty environment name", args: []string{"--update.a.path=/Root/Key", "--update.a.valueFromEnv="}, wantError: true},
		{name: "missing path", args: []string{"--update.a.value=hello"}, wantError: true},
		{name: "missing value", args: []string{"--update.a.path=/Root/Key"}, wantError: true},
		{name: "both sources", args: []string{"--update.a.path=/Root/Key", "--update.a.value=hello", "--update.a.valueFromEnv=TOKEN"}, wantError: true},
		{name: "environment fields", environ: []string{"CONFIGARR__UPDATE_A_PATH=/Root/Key", "CONFIGARR__UPDATE_A_VALUEFROMENV=TOKEN", "TOKEN=hello"}, want: "hello"},
		{name: "empty environment field", environ: []string{"CONFIGARR__UPDATE_A_PATH=/Root/Key", "CONFIGARR__UPDATE_A_VALUE="}, wantError: true},
		{name: "CLI precedence", args: []string{"--update.a.value=cli"}, environ: []string{"CONFIGARR__UPDATE_A_PATH=/Root/Key", "CONFIGARR__UPDATE_A_VALUE=env"}, want: "cli"},
		{name: "mixed sources conflict", args: []string{"--update.a.value=cli"}, environ: []string{"CONFIGARR__UPDATE_A_PATH=/Root/Key", "CONFIGARR__UPDATE_A_VALUEFROMENV=TOKEN", "TOKEN=secret"}, wantError: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			setEnvironment(t, tc.environ)
			args := append([]string{"--file", "unused.xml"}, tc.args...)

			options, err := cli.Parse(args)
			var replacements []xmlconfig.Update
			if err == nil {
				replacements, err = updates.Resolve(options.Updates)
			}

			if tc.wantError {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Len(t, replacements, 1)
			assert.Equal(t, tc.want, replacements[0].Value)
		})
	}
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

func TestDifferentValueSourcesAcrossUpdates(t *testing.T) {
	setEnvironment(t, []string{"TOKEN=environment"})
	args := []string{
		"--file=unused.xml",
		"--update.a.path=/Root/First", "--update.a.value=literal",
		"--update.b.path=/Root/Second", "--update.b.valueFromEnv=TOKEN",
	}
	options, err := cli.Parse(args)
	require.NoError(t, err)

	result, err := updates.Resolve(options.Updates)
	require.NoError(t, err)

	require.Len(t, result, 2)
	assert.Equal(t, "literal", result[0].Value)
	assert.Equal(t, "environment", result[1].Value)
}
