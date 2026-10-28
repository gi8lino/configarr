package updates_test

import (
	"os"
	"path/filepath"
	"testing"

	"configarr/internal/updates"
	"configarr/internal/xmlconfig"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestApplyReturnsFileResults(t *testing.T) {
	directory := t.TempDir()
	changed := filepath.Join(directory, "changed.xml")
	unchanged := filepath.Join(directory, "unchanged.xml")
	missing := filepath.Join(directory, "missing.xml")
	require.NoError(t, os.WriteFile(changed, []byte(`<Root><Key>old</Key></Root>`), 0600))
	require.NoError(t, os.WriteFile(unchanged, []byte(`<Root><Key>new</Key></Root>`), 0600))
	replacements := []xmlconfig.Update{{ID: "key", Path: "/Root/Key", Value: "new"}}

	results, err := updates.Apply([]string{changed, unchanged, missing}, replacements, true)
	require.NoError(t, err)

	assert.Equal(t, []updates.Result{
		{FilePath: changed, Changed: true},
		{FilePath: unchanged},
		{FilePath: missing, SkippedMissing: true},
	}, results)
	assert.NoFileExists(t, missing)
}

func TestApplyReturnsCompletedResultsOnFailure(t *testing.T) {
	directory := t.TempDir()
	first := filepath.Join(directory, "first.xml")
	missing := filepath.Join(directory, "missing.xml")
	last := filepath.Join(directory, "last.xml")
	original := []byte(`<Root><Key>old</Key></Root>`)
	require.NoError(t, os.WriteFile(first, original, 0600))
	require.NoError(t, os.WriteFile(last, original, 0600))
	replacements := []xmlconfig.Update{{ID: "key", Path: "/Root/Key", Value: "new"}}

	results, err := updates.Apply([]string{first, missing, last}, replacements, false)
	require.ErrorIs(t, err, os.ErrNotExist)

	assert.Contains(t, err.Error(), missing)
	assert.Equal(t, []updates.Result{{FilePath: first, Changed: true}}, results)
	content, err := os.ReadFile(last)
	require.NoError(t, err)
	assert.Equal(t, original, content)
}
