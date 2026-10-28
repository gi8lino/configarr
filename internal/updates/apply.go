package updates

import (
	"errors"
	"fmt"
	"os"

	"configarr/internal/xmlconfig"
)

// Result describes the outcome of applying replacements to one file.
type Result struct {
	// FilePath identifies the input file.
	FilePath string
	// Changed reports whether the file was updated.
	Changed bool
	// SkippedMissing reports a missing file skipped by the caller's policy.
	SkippedMissing bool
}

// Apply updates files without terminal output and returns completed results even if a later file fails.
func Apply(files []string, replacements []xmlconfig.Update, ignoreMissing bool) ([]Result, error) {
	var results []Result

	for _, file := range files {
		changed, err := xmlconfig.EditFile(file, replacements)
		if err != nil {
			if ignoreMissing && errors.Is(err, os.ErrNotExist) {
				results = append(
					results,
					Result{
						FilePath:       file,
						SkippedMissing: true,
					})
				continue
			}
			return results, fmt.Errorf("%s: %w", file, err)
		}

		results = append(
			results,
			Result{
				FilePath: file,
				Changed:  changed,
			},
		)
	}

	return results, nil
}
