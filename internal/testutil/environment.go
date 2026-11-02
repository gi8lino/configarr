// Package testutil provides isolated environments for parallel tests.
package testutil

import (
	"flag"
	"os"
	"os/exec"
	"regexp"
	"slices"
	"strings"
	"testing"
)

const environmentTest = "CONFIGARR_TEST_PROCESS"

// WithEnvironment reruns the current test in an isolated process and reports whether its body should execute.
func WithEnvironment(t *testing.T, entries []string) bool {
	t.Helper()

	if os.Getenv(environmentTest) == t.Name() {
		return true
	}

	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}

	segments := strings.Split(t.Name(), "/")
	for i, segment := range segments {
		segments[i] = "^" + regexp.QuoteMeta(segment) + "$"
	}
	command := exec.CommandContext(t.Context(), executable, "-test.run="+strings.Join(segments, "/"))
	// Share coverage output so assertions executed in children remain represented.
	if coverage := flag.Lookup("test.gocoverdir"); coverage != nil && coverage.Value.String() != "" {
		command.Args = append(command.Args, "-test.gocoverdir="+coverage.Value.String())
	}
	command.Env = isolatedEnvironment(entries)
	command.Env = append(command.Env, environmentTest+"="+t.Name())

	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("isolated test failed: %v\n%s", err, output)
	}
	return false
}

// isolatedEnvironment replaces application and fixture variables while preserving the host runtime environment.
func isolatedEnvironment(entries []string) []string {
	var environment []string
	for _, entry := range os.Environ() {
		key, _, _ := strings.Cut(entry, "=")
		if strings.HasPrefix(key, "CONFIGARR_") || slices.Contains([]string{"TOKEN", "SECRET", "DATABASE_HOST"}, key) {
			continue
		}
		environment = append(environment, entry)
	}
	return append(environment, entries...)
}
