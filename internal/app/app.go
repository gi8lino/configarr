package app

import (
	"fmt"
	"io"

	"configarr/internal/cli"
	"configarr/internal/updates"

	"github.com/containeroo/tinyflags"
)

// Run parses the invocation and applies generic XML updates from explicit dynamic requests.
func Run(args []string, stdout, stderr io.Writer) error {
	options, err := cli.Parse(args)
	if err != nil {
		if tinyflags.IsHelpRequested(err) || tinyflags.IsVersionRequested(err) {
			fmt.Fprint(stdout, err.Error()) // nolint:errcheck
			return nil
		}
		fmt.Fprintln(stderr, err) // nolint:errcheck

		return err
	}

	replacements, err := updates.Resolve(options.Updates)
	if err != nil {
		fmt.Fprintln(stderr, err) // nolint:errcheck
		return err
	}

	results, err := updates.Apply(options.FilePaths, replacements, options.IgnoreMissingConfig)
	for _, result := range results {
		switch {
		case result.SkippedMissing:
			fmt.Fprintln(stderr, result.FilePath+": no configuration file found. Skipping update.") // nolint:errcheck
		case result.Changed:
			fmt.Fprintln(stdout, result.FilePath+": updated XML configuration", "updates", len(replacements)) // nolint:errcheck
		default:
			fmt.Fprintln(stdout, result.FilePath+": no updates made to the configuration.") // nolint:errcheck
		}
	}

	if err != nil {
		fmt.Fprintln(stderr, err) // nolint:errcheck
	}
	return err
}
