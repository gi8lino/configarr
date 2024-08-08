package main

import (
	"bytes"
	"encoding/xml"
	"log/slog"
	"os"
	"strings"
	"testing"
)

// TestConfig_UnmarshalXML tests the XML unmarshalling into Config struct.
func TestConfig_UnmarshalXML(t *testing.T) {
	t.Run("Valid XML", func(t *testing.T) {
		xmlData := `<Config><LogLevel>info</LogLevel><Theme>dark</Theme></Config>`
		var config Config
		err := xml.Unmarshal([]byte(xmlData), &config)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if config.Properties["LogLevel"] != "info" || config.Properties["Theme"] != "dark" {
			t.Fatalf("Expected properties not set correctly: %v", config.Properties)
		}

		if len(config.Keys) != 2 || config.Keys[0] != "LogLevel" || config.Keys[1] != "Theme" {
			t.Fatalf("Expected key order ['LogLevel', 'Theme'], got %v", config.Keys)
		}
	})

	t.Run("Invalid XML", func(t *testing.T) {
		xmlData := `<Config><LogLevel>info<LogLevel></Config>` // malformed XML
		var config Config
		err := xml.Unmarshal([]byte(xmlData), &config)
		if err == nil {
			t.Fatal("Expected error due to malformed XML, but got none")
		}
	})
}

// TestConfig_MarshalXML tests the XML marshalling from Config struct.
func TestConfig_MarshalXML(t *testing.T) {
	t.Run("Marshal to XML", func(t *testing.T) {
		config := Config{
			Properties: map[string]string{
				"LogLevel": "info",
				"Theme":    "dark",
			},
			Keys: []string{"LogLevel", "Theme"},
		}

		expectedXML := `<Config><LogLevel>info</LogLevel><Theme>dark</Theme></Config>`
		output, err := xml.Marshal(&config)
		if err != nil {
			t.Fatalf("Unexpected error during marshalling: %v", err)
		}

		output = bytes.TrimSpace(output) // Trim space to match expected exactly
		if string(output) != expectedXML {
			t.Fatalf("Expected XML %s, got %s", expectedXML, output)
		}
	})
}

// TestReadAndParseXML tests the reading and parsing of XML from a file.
func TestReadAndParseXML(t *testing.T) {
	t.Run("File Exists and Valid XML", func(t *testing.T) {
		content := `<Config><LogLevel>info</LogLevel><Theme>dark</Theme></Config>`
		file, err := os.CreateTemp("", "test*.xml")
		if err != nil {
			t.Fatalf("Unexpected error creating temp file: %v", err)
		}
		defer os.Remove(file.Name())

		if _, err := file.Write([]byte(content)); err != nil {
			t.Fatalf("Unexpected error writing to temp file: %v", err)
		}
		file.Close()

		config, err := readAndParseXML(file.Name())
		if err != nil {
			t.Fatalf("Unexpected error reading XML: %v", err)
		}

		if config.Properties["LogLevel"] != "info" || config.Properties["Theme"] != "dark" {
			t.Fatalf("Expected properties not set correctly: %v", config.Properties)
		}

		if len(config.Keys) != 2 || config.Keys[0] != "LogLevel" || config.Keys[1] != "Theme" {
			t.Fatalf("Expected key order ['LogLevel', 'Theme'], got %v", config.Keys)
		}
	})

	t.Run("File Does Not Exist", func(t *testing.T) {
		_, err := readAndParseXML("nonexistent.xml")
		if err == nil {
			t.Fatal("Expected error for nonexistent file, got none")
		}
	})
}

// TestUpdateConfigWithEnv tests updating configuration with environment variables.
func TestUpdateConfigWithEnv(t *testing.T) {
	t.Run("Update with Environment Variables", func(t *testing.T) {
		envVars := []string{
			"CONFIGARR__LOG=LogLevel=debug",
			"CONFIGARR__THEME=Theme=light",
		}

		config := &Config{
			Properties: map[string]string{
				"LogLevel": "info",
				"Theme":    "dark",
			},
			Keys: []string{"LogLevel", "Theme"},
		}

		var stdOut strings.Builder
		logger := slog.New(slog.NewTextHandler(&stdOut, &slog.HandlerOptions{Level: slog.LevelDebug}))

		changed := updateConfigWithEnv(envVars, config, "CONFIGARR__", logger)
		if len(changed) != 2 || changed["LogLevel"] != "debug" || changed["Theme"] != "light" {
			t.Fatalf("Expected changes not applied correctly: %v", changed)
		}

		if !strings.Contains(stdOut.String(), "Updated 'LogLevel' to 'debug'") {
			t.Fatalf("Expected log entry for LogLevel change, got: %s", stdOut.String())
		}
	})

	t.Run("No Changes When Env Vars Unmatched", func(t *testing.T) {
		envVars := []string{
			"OTHER_LOG=LogLevel=debug",
		}

		config := &Config{
			Properties: map[string]string{
				"LogLevel": "info",
			},
			Keys: []string{"LogLevel"},
		}

		var stdOut strings.Builder
		logger := slog.New(slog.NewTextHandler(&stdOut, &slog.HandlerOptions{Level: slog.LevelDebug}))

		changed := updateConfigWithEnv(envVars, config, "CONFIGARR__", logger)
		if len(changed) != 0 {
			t.Fatalf("Expected no changes, but got: %v", changed)
		}

		if !strings.Contains(stdOut.String(), "No updates made to the configuration.") {
			t.Fatalf("Expected log entry for no updates, got: %s", stdOut.String())
		}
	})
}

// TestWriteConfigToFile tests writing the configuration back to the XML file.
func TestWriteConfigToFile(t *testing.T) {
	t.Run("Write to XML File", func(t *testing.T) {
		config := &Config{
			Properties: map[string]string{
				"Theme":    "dark",
				"LogLevel": "info",
			},
			Keys: []string{"Theme", "LogLevel"},
		}

		file, err := os.CreateTemp("", "test*.xml")
		if err != nil {
			t.Fatalf("Unexpected error creating temp file: %v", err)
		}
		defer os.Remove(file.Name())

		if err := writeConfigToFile(config, file.Name()); err != nil {
			t.Fatalf("Unexpected error writing to XML file: %v", err)
		}

		content, err := os.ReadFile(file.Name())
		if err != nil {
			t.Fatalf("Unexpected error reading written file: %v", err)
		}

		expectedXML := `<Config>
  <Theme>dark</Theme>
  <LogLevel>info</LogLevel>
</Config>`
		if string(content) != expectedXML {
			t.Fatalf("Expected XML %s, got %s", expectedXML, string(content))
		}
	})
}

// TestParseFlags tests the parsing of command-line flags.
func TestParseFlags(t *testing.T) {
	t.Run("Parse valid flags", func(t *testing.T) {
		args := []string{"--config", "/path/to/config.xml", "--prefix", "PREFIX__", "--debug", "--ignore-missing-config"}
		expectedFlags := Flags{
			ConfigFilePath:      "/path/to/config.xml",
			Prefix:              "PREFIX__",
			Debug:               true,
			IgnoreMissingConfig: true,
		}

		flags, err := parseFlags(args)
		if err != nil {
			t.Fatalf("Unexpected error parsing flags: %v", err)
		}

		if flags != expectedFlags {
			t.Fatalf("Expected flags %+v, got %+v", expectedFlags, flags)
		}
	})

	t.Run("Error on invalid flags", func(t *testing.T) {
		args := []string{"--invalid"}
		_, err := parseFlags(args)
		if err == nil {
			t.Fatal("Expected error on invalid flags, but got none")
		}
	})
}

// TestRun tests the main functionality of the application, ensuring it updates
// the configuration file based on environment variables.
func TestRun(t *testing.T) {
	t.Run("Update Configuration and Write", func(t *testing.T) {
		// Set up temporary XML file
		xmlContent := `<Config>
  <LogLevel>info</LogLevel>
  <Theme>dark</Theme>
</Config>`
		file, err := os.CreateTemp("", "config*.xml")
		if err != nil {
			t.Fatalf("Unexpected error creating temp file: %v", err)
		}
		defer os.Remove(file.Name())

		if _, err := file.Write([]byte(xmlContent)); err != nil {
			t.Fatalf("Unexpected error writing XML content to temp file: %v", err)
		}
		file.Close()

		envVars := []string{
			"CONFIGARR__LOG=LogLevel=debug",
		}

		// Prepare arguments to simulate command-line input
		args := []string{"cmd", "--config", file.Name(), "--prefix", "CONFIGARR__", "--debug"}

		var stdOut strings.Builder
		err = run(envVars, args, &stdOut)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		// Read the updated file
		updatedContent, err := os.ReadFile(file.Name())
		if err != nil {
			t.Fatalf("Unexpected error reading updated file: %v", err)
		}

		expectedXML := `<Config>
  <LogLevel>debug</LogLevel>
  <Theme>dark</Theme>
</Config>`
		if string(updatedContent) != expectedXML {
			t.Fatalf("Expected XML %s, got %s", expectedXML, string(updatedContent))
		}

		if !strings.Contains(stdOut.String(), "Updated 'LogLevel' to 'debug'") {
			t.Fatalf("Expected log entry for LogLevel change, got: %s", stdOut.String())
		}
	})

	t.Run("No updates when environment variables do not match", func(t *testing.T) {
		// Set up temporary XML file
		xmlContent := `<Config>
  <LogLevel>info</LogLevel>
  <Theme>dark</Theme>
</Config>`
		file, err := os.CreateTemp("", "config*.xml")
		if err != nil {
			t.Fatalf("Unexpected error creating temp file: %v", err)
		}
		defer os.Remove(file.Name())

		if _, err := file.Write([]byte(xmlContent)); err != nil {
			t.Fatalf("Unexpected error writing XML content to temp file: %v", err)
		}
		file.Close()

		envVars := []string{
			"OTHER__LEVEL=LogLevel=debug",
		}

		// Prepare arguments to simulate command-line input
		args := []string{"cmd", "--config", file.Name(), "--prefix", "CONFIGARR__", "--debug"}

		var stdOut strings.Builder
		err = run(envVars, args, &stdOut)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		// Read the file to ensure it remains unchanged
		updatedContent, err := os.ReadFile(file.Name())
		if err != nil {
			t.Fatalf("Unexpected error reading file: %v", err)
		}

		expectedXML := `<Config>
  <LogLevel>info</LogLevel>
  <Theme>dark</Theme>
</Config>`
		if string(updatedContent) != expectedXML {
			t.Fatalf("Expected no changes, but got different XML: %s", string(updatedContent))
		}

		if !strings.Contains(stdOut.String(), "No updates made to the configuration.") {
			t.Fatalf("Expected log entry for no updates, got: %s", stdOut.String())
		}
	})

	t.Run("Ignore missing config file", func(t *testing.T) {
		// Ensure the file does not exist
		nonExistentFile := "nonexistent.xml"

		envVars := []string{
			"CONFIGARR__LOG=LogLevel=debug",
		}

		// Prepare arguments to simulate command-line input
		args := []string{"cmd", "--config", nonExistentFile, "--prefix", "CONFIGARR__", "--ignore-missing-config", "--debug"}

		var stdOut strings.Builder
		err := run(envVars, args, &stdOut)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if !strings.Contains(stdOut.String(), "No configuration file found. Skipping update.") {
			t.Fatalf("Expected log entry for skipping update due to missing file, got: %s", stdOut.String())
		}
	})

	t.Run("Error when config file is missing without ignore flag", func(t *testing.T) {
		// Ensure the file does not exist
		nonExistentFile := "nonexistent.xml"

		envVars := []string{
			"CONFIGARR__LOG=LogLevel=debug",
		}

		// Prepare arguments to simulate command-line input
		args := []string{"cmd", "--config", nonExistentFile, "--prefix", "CONFIGARR__"}

		var logOutput bytes.Buffer
		err := run(envVars, args, &logOutput)
		if err == nil {
			t.Fatal("Expected error for missing configuration file, but got none")
		}
	})
}
