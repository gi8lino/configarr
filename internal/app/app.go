package app

import (
	"errors"
	"fmt"
	"io"

	"configarr/internal/cli"
	"configarr/internal/report"
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
	reportErr := report.Write(stdout, stderr, results, options.Output, options.ShowValues)
	err = errors.Join(err, reportErr)

	if err != nil {
		fmt.Fprintln(stderr, err) // nolint:errcheck
	}
	return err
}
