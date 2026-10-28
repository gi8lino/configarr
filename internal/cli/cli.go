package cli

import (
	"fmt"
	"slices"

	"configarr/internal/updates"
	"github.com/containeroo/tinyflags"
)

// Options contains application settings and parsed update fields.
type Options struct {
	// FilePaths lists the XML files that receive the same updates.
	FilePaths []string
	// IgnoreMissingConfig allows a missing file to be skipped.
	IgnoreMissingConfig bool
	// Updates retains raw dynamic fields for the update resolver.
	Updates updates.Inputs
}

// Parse delegates flag and environment parsing to tinyflags without resolving replacement values.
func Parse(args []string) (Options, error) {
	flagSet := tinyflags.NewFlagSet("configarr", tinyflags.ContinueOnError)
	flagSet.EnvPrefix("CONFIGARR_")
	flagSet.Note("Pass XML filenames as positional arguments or repeat --file. Use -- before filenames that begin with a dash.")

	options := Options{}
	files := flagSet.StringSlice("file", nil, "XML file to update; may be repeated").
		NoSplit().
		PreserveSpace().
		Value()
	flagSet.BoolVar(&options.IgnoreMissingConfig, "ignore-missing-config", false, "Ignore missing configuration file")

	group := flagSet.DynamicGroup("update")
	paths := group.String("path", "", "Absolute XML element or attribute path")
	paths.Required().NotEmpty()

	values := group.String("value", "", "Literal replacement text (may be empty)")
	envNames := group.String("valueFromEnv", "", "Environment variable containing replacement text")
	envNames.NotEmpty()

	group.OneOf(values, envNames).Required()

	flagSet.Validate(func() error {
		options.FilePaths = selectFilePaths(*files, flagSet.Args(), flagSet.Origin("file").Source)
		if len(options.FilePaths) == 0 {
			return fmt.Errorf("provide at least one XML file as a positional argument or with --file")
		}
		if slices.Contains(options.FilePaths, "") {
			return fmt.Errorf("XML file paths must not be empty")
		}

		return nil
	})

	if err := flagSet.Parse(args); err != nil {
		return Options{}, err
	}

	options.Updates = updates.Inputs{
		Paths:        paths.Values(),
		Values:       values.Values(),
		ValueFromEnv: envNames.Values(),
	}

	return options, nil
}

// selectFilePaths combines explicit file arguments and uses the environment only as a fallback.
func selectFilePaths(flagPaths, positional []string, source tinyflags.ValueSource) []string {
	if source == tinyflags.ValueSourceEnvironment {
		if len(positional) > 0 {
			return positional
		}
	}
	return append(flagPaths, positional...)
}
