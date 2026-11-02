// Package report formats completed XML updates for people and scripts.
package report

import (
	"encoding/json"
	"fmt"
	"io"
	"strconv"

	"configarr/internal/updates"
)

// Format selects how completed updates are rendered.
type Format string

const (
	// Text renders one human-readable line per change or file outcome.
	Text Format = "text"
	// JSON renders one JSON object per line.
	JSON Format = "json"
)

// record describes one change or a file that needed no changes.
type record struct {
	// File identifies the input file.
	File string `json:"file"`
	// Status is updated, unchanged, or skipped.
	Status string `json:"status"`
	// Path identifies the changed XML selector, when present.
	Path string `json:"path,omitempty"`
	// Before includes the original value only when explicitly requested.
	Before *string `json:"before,omitempty"`
	// After includes the replacement value only when explicitly requested.
	After *string `json:"after,omitempty"`
}

// Write reports each effective change or unchanged/skipped file in the selected format.
func Write(stdout, stderr io.Writer, results []updates.Result, format Format, showValues bool) error {
	for _, result := range results {
		entry := record{
			File:   result.FilePath,
			Status: "unchanged",
		}

		if result.SkippedMissing {
			entry.Status = "skipped"
		}

		if len(result.Changes) == 0 {
			if err := writeRecord(stdout, stderr, entry, format); err != nil {
				return err
			}
			continue
		}

		for _, change := range result.Changes {
			entry.Status = "updated"
			entry.Path = change.Path
			if showValues {
				entry.Before = &change.Before
				entry.After = &change.After
			}
			if err := writeRecord(stdout, stderr, entry, format); err != nil {
				return err
			}
		}
	}
	return nil
}

// writeRecord emits one complete line, keeping JSON records together on stdout.
func writeRecord(stdout, stderr io.Writer, entry record, format Format) error {
	if format == JSON {
		return json.NewEncoder(stdout).Encode(entry)
	}

	file := singleLine(entry.File)
	switch entry.Status {
	case "skipped":
		_, err := fmt.Fprintln(stderr, file+": no configuration file found. Skipping update.")
		return err
	case "unchanged":
		_, err := fmt.Fprintln(stdout, file+": no updates made to the configuration.")
		return err
	default:
		if entry.Before != nil {
			_, err := fmt.Fprintf(stdout, "%s: %s: %q -> %q\n", file, singleLine(entry.Path), *entry.Before, *entry.After)
			return err
		}
		_, err := fmt.Fprintf(stdout, "%s: %s: updated\n", file, singleLine(entry.Path))
		return err
	}
}

// singleLine escapes control characters while leaving ordinary filenames and selectors readable.
func singleLine(value string) string {
	quoted := strconv.Quote(value)
	return quoted[1 : len(quoted)-1]
}
