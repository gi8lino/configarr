package xmlconfig

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// EditFile streams targeted changes into a temporary file and atomically replaces the resolved target.
func EditFile(path string, updates []Update) (bool, error) {
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		return false, fmt.Errorf("resolve XML file: %w", err)
	}
	source, err := os.Open(resolved)
	if err != nil {
		return false, fmt.Errorf("open XML file: %w", err)
	}
	defer closeSourceOnExit(source)

	info, err := source.Stat()
	if err != nil {
		return false, err
	}
	if !info.Mode().IsRegular() {
		return false, fmt.Errorf("XML input must be a regular file")
	}
	edits, err := planEdits(source, info.Size(), updates)
	if err != nil {
		return false, err
	}
	if len(edits) == 0 {
		return false, nil
	}

	temporary, err := os.CreateTemp(filepath.Dir(resolved), ".configarr-*")
	if err != nil {
		return false, fmt.Errorf("create temporary XML file: %w", err)
	}
	defer discardTemporaryFile(temporary)

	if err := copyEdits(source, info.Size(), temporary, edits); err != nil {
		return false, fmt.Errorf("write XML updates: %w", err)
	}

	if err := preserveOwnership(temporary, info); err != nil {
		return false, fmt.Errorf("preserve XML ownership: %w", err)
	}
	if err := temporary.Chmod(info.Mode()); err != nil {
		return false, fmt.Errorf("preserve XML permissions: %w", err)
	}

	if err := temporary.Sync(); err != nil {
		return false, fmt.Errorf("sync XML updates: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return false, err
	}
	if err := source.Close(); err != nil {
		return false, err
	}

	if err := os.Rename(temporary.Name(), resolved); err != nil {
		return false, fmt.Errorf("replace XML file: %w", err)
	}

	return true, nil
}

// copyEdits copies untouched byte ranges through a fixed-size buffer between replacements.
func copyEdits(source io.ReaderAt, size int64, output io.Writer, edits []replacement) error {
	buffer := make([]byte, 32*1024)
	position := int64(0)

	for _, edit := range edits {
		if edit.region.start < position {
			return fmt.Errorf("overlapping XML edits")
		}

		if _, err := io.CopyBuffer(output, io.NewSectionReader(source, position, edit.region.start-position), buffer); err != nil {
			return err
		}

		written, err := output.Write(edit.value)
		if err != nil {
			return err
		}
		if written != len(edit.value) {
			return io.ErrShortWrite
		}

		position = edit.region.end
	}

	_, err := io.CopyBuffer(output, io.NewSectionReader(source, position, size-position), buffer)
	return err
}

// closeSourceOnExit releases the source on early returns after preserving any primary error.
func closeSourceOnExit(source *os.File) {
	_ = source.Close()
}

// discardTemporaryFile closes and removes staging data left after failure or successful replacement.
func discardTemporaryFile(temporary *os.File) {
	_ = temporary.Close()
	_ = os.Remove(temporary.Name())
}
