package updates_test

import (
	"testing"

	"configarr/internal/cli"
	"configarr/internal/testutil"
	"configarr/internal/updates"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolve(t *testing.T) {
	t.Parallel()

	t.Run("literal", func(t *testing.T) {
		t.Parallel()

		if !testutil.WithEnvironment(t, nil) {
			return
		}
		args := []string{
			"--file", "unused.xml",
			"--update.a.path=/Root/Key",
			"--update.a.value=hello=world",
		}

		options, err := cli.Parse(args)
		require.NoError(t, err)

		replacements, err := updates.Resolve(options.Updates)
		require.NoError(t, err)
		require.Len(t, replacements, 1)
		assert.Equal(t, "hello=world", replacements[0].Value)
	})

	t.Run("empty literal", func(t *testing.T) {
		t.Parallel()

		if !testutil.WithEnvironment(t, nil) {
			return
		}
		args := []string{
			"--file", "unused.xml",
			"--update.a.path=/Root/Key",
			"--update.a.value=",
		}

		options, err := cli.Parse(args)
		require.NoError(t, err)

		replacements, err := updates.Resolve(options.Updates)
		require.NoError(t, err)
		require.Len(t, replacements, 1)
		assert.Equal(t, "", replacements[0].Value)
	})

	t.Run("environment source", func(t *testing.T) {
		t.Parallel()

		if !testutil.WithEnvironment(t, []string{"TOKEN= secret=&value "}) {
			return
		}
		args := []string{
			"--file", "unused.xml",
			"--update.a.path=/Root/Key",
			"--update.a.valueFromEnv=TOKEN",
		}

		options, err := cli.Parse(args)
		require.NoError(t, err)

		replacements, err := updates.Resolve(options.Updates)
		require.NoError(t, err)
		require.Len(t, replacements, 1)
		assert.Equal(t, " secret=&value ", replacements[0].Value)
	})

	t.Run("empty environment source", func(t *testing.T) {
		t.Parallel()

		if !testutil.WithEnvironment(t, []string{"TOKEN="}) {
			return
		}
		args := []string{
			"--file", "unused.xml",
			"--update.a.path=/Root/Key",
			"--update.a.valueFromEnv=TOKEN",
		}

		options, err := cli.Parse(args)
		require.NoError(t, err)

		replacements, err := updates.Resolve(options.Updates)
		require.NoError(t, err)
		require.Len(t, replacements, 1)
		assert.Equal(t, "", replacements[0].Value)
	})

	t.Run("unset environment source", func(t *testing.T) {
		t.Parallel()

		if !testutil.WithEnvironment(t, nil) {
			return
		}
		args := []string{
			"--file", "unused.xml",
			"--update.a.path=/Root/Key",
			"--update.a.valueFromEnv=TOKEN",
		}

		options, err := cli.Parse(args)
		require.NoError(t, err)

		_, err = updates.Resolve(options.Updates)

		assert.Error(t, err)
	})

	t.Run("environment fields", func(t *testing.T) {
		t.Parallel()

		if !testutil.WithEnvironment(t, []string{
			"CONFIGARR__UPDATE_A_PATH=/Root/Key",
			"CONFIGARR__UPDATE_A_VALUEFROMENV=TOKEN",
			"TOKEN=hello",
		}) {
			return
		}
		args := []string{
			"--file", "unused.xml",
		}

		options, err := cli.Parse(args)
		require.NoError(t, err)

		replacements, err := updates.Resolve(options.Updates)
		require.NoError(t, err)
		require.Len(t, replacements, 1)
		assert.Equal(t, "hello", replacements[0].Value)
	})

	t.Run("CLI precedence", func(t *testing.T) {
		t.Parallel()

		if !testutil.WithEnvironment(t, []string{
			"CONFIGARR__UPDATE_A_PATH=/Root/Key",
			"CONFIGARR__UPDATE_A_VALUE=env",
		}) {
			return
		}
		args := []string{
			"--file", "unused.xml",
			"--update.a.value=cli",
		}

		options, err := cli.Parse(args)
		require.NoError(t, err)

		replacements, err := updates.Resolve(options.Updates)
		require.NoError(t, err)
		require.Len(t, replacements, 1)
		assert.Equal(t, "cli", replacements[0].Value)
	})

	t.Run("different value sources across updates", func(t *testing.T) {
		t.Parallel()

		if !testutil.WithEnvironment(t, []string{"TOKEN=environment"}) {
			return
		}
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
	})
}
