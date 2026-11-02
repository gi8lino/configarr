package cli

import (
	"testing"

	"configarr/internal/report"
	"configarr/internal/testutil"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParse(t *testing.T) {
	t.Parallel()

	t.Run("parses output settings", func(t *testing.T) {
		t.Parallel()

		if !testutil.WithEnvironment(t, nil) {
			return
		}
		options, err := Parse([]string{"config.xml", "--output=json", "--show-values"})
		require.NoError(t, err)
		assert.Equal(t, report.JSON, options.Output)
		assert.True(t, options.ShowValues)
	})

	t.Run("rejects unsupported output format", func(t *testing.T) {
		t.Parallel()

		if !testutil.WithEnvironment(t, nil) {
			return
		}
		_, err := Parse([]string{"config.xml", "--output=yaml"})
		assert.Error(t, err)
	})

	t.Run("valid flags", func(t *testing.T) {
		t.Parallel()

		if !testutil.WithEnvironment(t, nil) {
			return
		}
		args := []string{"--file", "/path/to/config.xml", "--ignore-missing-config"}

		options, err := Parse(args)
		require.NoError(t, err)

		assert.Equal(t, []string{"/path/to/config.xml"}, options.FilePaths)
		assert.True(t, options.IgnoreMissingConfig)
		assert.Empty(t, options.Updates.Paths)
	})

	t.Run("invalid flags", func(t *testing.T) {
		t.Parallel()

		if !testutil.WithEnvironment(t, nil) {
			return
		}
		_, err := Parse([]string{"--invalid"})
		assert.Error(t, err)
	})

	t.Run("environment defaults", func(t *testing.T) {
		t.Parallel()

		if !testutil.WithEnvironment(t, []string{"CONFIGARR__FILE=from-env.xml", "CONFIGARR__IGNORE_MISSING_CONFIG=true"}) {
			return
		}

		options, err := Parse(nil)
		require.NoError(t, err)

		assert.Equal(t, []string{"from-env.xml"}, options.FilePaths)
		assert.True(t, options.IgnoreMissingConfig)
	})

	t.Run("flag overrides environment default", func(t *testing.T) {
		t.Parallel()

		if !testutil.WithEnvironment(t, []string{"CONFIGARR__FILE=from-env.xml", "CONFIGARR__IGNORE_MISSING_CONFIG=true"}) {
			return
		}

		options, err := Parse([]string{"--file=from-cli.xml"})
		require.NoError(t, err)

		assert.Equal(t, []string{"from-cli.xml"}, options.FilePaths)
		assert.True(t, options.IgnoreMissingConfig)
	})

	t.Run("keeps environment value source unresolved", func(t *testing.T) {
		t.Parallel()

		if !testutil.WithEnvironment(t, nil) {
			return
		}
		args := []string{"--file=unused.xml", "--update.a.path=/Root/Key", "--update.a.valueFromEnv=UNSET_SOURCE"}

		options, err := Parse(args)
		require.NoError(t, err)

		assert.Equal(t, "UNSET_SOURCE", options.Updates.ValueFromEnv["a"])
	})

	t.Run("rejects removed prefix flag", func(t *testing.T) {
		t.Parallel()

		if !testutil.WithEnvironment(t, nil) {
			return
		}

		_, err := Parse([]string{"--file=unused.xml", "--prefix=OLD__"})
		assert.Error(t, err)
	})

	t.Run("one positional file", func(t *testing.T) {
		t.Parallel()

		if !testutil.WithEnvironment(t, nil) {
			return
		}

		options, err := Parse([]string{"radarr.xml"})
		require.NoError(t, err)

		assert.Equal(t, []string{"radarr.xml"}, options.FilePaths)
	})

	t.Run("multiple positional files", func(t *testing.T) {
		t.Parallel()

		if !testutil.WithEnvironment(t, nil) {
			return
		}

		options, err := Parse([]string{"radarr.xml", "sonarr.xml"})
		require.NoError(t, err)

		assert.Equal(t, []string{"radarr.xml", "sonarr.xml"}, options.FilePaths)
	})

	t.Run("repeated file flag", func(t *testing.T) {
		t.Parallel()

		if !testutil.WithEnvironment(t, nil) {
			return
		}

		options, err := Parse([]string{"--file=radarr.xml", "--file", "sonarr.xml"})
		require.NoError(t, err)

		assert.Equal(t, []string{"radarr.xml", "sonarr.xml"}, options.FilePaths)
	})

	t.Run("combined forms", func(t *testing.T) {
		t.Parallel()

		if !testutil.WithEnvironment(t, nil) {
			return
		}

		options, err := Parse([]string{"sonarr.xml", "--file=radarr.xml"})
		require.NoError(t, err)

		assert.Equal(t, []string{"radarr.xml", "sonarr.xml"}, options.FilePaths)
	})

	t.Run("intermixed flags", func(t *testing.T) {
		t.Parallel()

		if !testutil.WithEnvironment(t, nil) {
			return
		}

		options, err := Parse([]string{"radarr.xml", "--ignore-missing-config", "sonarr.xml"})
		require.NoError(t, err)

		assert.Equal(t, []string{"radarr.xml", "sonarr.xml"}, options.FilePaths)
	})

	t.Run("literal filename", func(t *testing.T) {
		t.Parallel()

		if !testutil.WithEnvironment(t, nil) {
			return
		}

		options, err := Parse([]string{"--file= radarr, sonarr.xml "})
		require.NoError(t, err)

		assert.Equal(t, []string{" radarr, sonarr.xml "}, options.FilePaths)
	})

	t.Run("dash filename", func(t *testing.T) {
		t.Parallel()

		if !testutil.WithEnvironment(t, nil) {
			return
		}

		options, err := Parse([]string{"--", "-radarr.xml"})
		require.NoError(t, err)

		assert.Equal(t, []string{"-radarr.xml"}, options.FilePaths)
	})

	t.Run("environment fallback", func(t *testing.T) {
		t.Parallel()

		if !testutil.WithEnvironment(t, []string{"CONFIGARR__FILE=radarr.xml"}) {
			return
		}

		options, err := Parse(nil)
		require.NoError(t, err)

		assert.Equal(t, []string{"radarr.xml"}, options.FilePaths)
	})

	t.Run("flag overrides environment", func(t *testing.T) {
		t.Parallel()

		if !testutil.WithEnvironment(t, []string{"CONFIGARR__FILE=radarr.xml"}) {
			return
		}

		options, err := Parse([]string{"--file=sonarr.xml"})
		require.NoError(t, err)

		assert.Equal(t, []string{"sonarr.xml"}, options.FilePaths)
	})

	t.Run("positional overrides environment", func(t *testing.T) {
		t.Parallel()

		if !testutil.WithEnvironment(t, []string{"CONFIGARR__FILE=radarr.xml"}) {
			return
		}

		options, err := Parse([]string{"sonarr.xml"})
		require.NoError(t, err)

		assert.Equal(t, []string{"sonarr.xml"}, options.FilePaths)
	})

	t.Run("no files", func(t *testing.T) {
		t.Parallel()

		if !testutil.WithEnvironment(t, nil) {
			return
		}

		_, err := Parse(nil)
		assert.Error(t, err)
	})

	t.Run("empty flag", func(t *testing.T) {
		t.Parallel()

		if !testutil.WithEnvironment(t, nil) {
			return
		}

		_, err := Parse([]string{"--file="})
		assert.Error(t, err)
	})

	t.Run("empty positional file", func(t *testing.T) {
		t.Parallel()

		if !testutil.WithEnvironment(t, nil) {
			return
		}

		_, err := Parse([]string{""})
		assert.Error(t, err)
	})

	t.Run("old flag rejected", func(t *testing.T) {
		t.Parallel()

		if !testutil.WithEnvironment(t, nil) {
			return
		}

		_, err := Parse([]string{"--config=radarr.xml"})
		assert.Error(t, err)
	})

	t.Run("old environment ignored", func(t *testing.T) {
		t.Parallel()

		if !testutil.WithEnvironment(t, []string{"CONFIGARR__CONFIG=radarr.xml"}) {
			return
		}

		_, err := Parse(nil)
		assert.Error(t, err)
	})

	t.Run("empty environment name", func(t *testing.T) {
		t.Parallel()

		if !testutil.WithEnvironment(t, nil) {
			return
		}
		args := []string{
			"--file", "unused.xml",
			"--update.a.path=/Root/Key",
			"--update.a.valueFromEnv=",
		}

		_, err := Parse(args)
		assert.Error(t, err)
	})

	t.Run("missing path", func(t *testing.T) {
		t.Parallel()

		if !testutil.WithEnvironment(t, nil) {
			return
		}
		args := []string{
			"--file", "unused.xml",
			"--update.a.value=hello",
		}

		_, err := Parse(args)
		assert.Error(t, err)
	})

	t.Run("missing value", func(t *testing.T) {
		t.Parallel()

		if !testutil.WithEnvironment(t, nil) {
			return
		}
		args := []string{
			"--file", "unused.xml",
			"--update.a.path=/Root/Key",
		}

		_, err := Parse(args)
		assert.Error(t, err)
	})

	t.Run("both sources", func(t *testing.T) {
		t.Parallel()

		if !testutil.WithEnvironment(t, nil) {
			return
		}
		args := []string{
			"--file", "unused.xml",
			"--update.a.path=/Root/Key",
			"--update.a.value=hello",
			"--update.a.valueFromEnv=TOKEN",
		}

		_, err := Parse(args)
		assert.Error(t, err)
	})

	t.Run("empty environment field", func(t *testing.T) {
		t.Parallel()

		if !testutil.WithEnvironment(t, []string{
			"CONFIGARR__UPDATE_A_PATH=/Root/Key",
			"CONFIGARR__UPDATE_A_VALUE=",
		}) {
			return
		}
		args := []string{
			"--file", "unused.xml",
		}

		_, err := Parse(args)
		assert.Error(t, err)
	})

	t.Run("mixed sources conflict", func(t *testing.T) {
		t.Parallel()

		if !testutil.WithEnvironment(t, []string{
			"CONFIGARR__UPDATE_A_PATH=/Root/Key",
			"CONFIGARR__UPDATE_A_VALUEFROMENV=TOKEN",
			"TOKEN=secret",
		}) {
			return
		}
		args := []string{
			"--file", "unused.xml",
			"--update.a.value=cli",
		}

		_, err := Parse(args)
		assert.Error(t, err)
	})
}
