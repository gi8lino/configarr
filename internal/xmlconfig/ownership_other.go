//go:build !unix

package xmlconfig

import "os"

// preserveOwnership leaves ownership to the operating system on platforms without Unix file owners.
func preserveOwnership(file *os.File, original os.FileInfo) error {
	return nil
}
