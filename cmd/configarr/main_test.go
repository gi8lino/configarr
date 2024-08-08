package main

import (
	"bytes"
	"encoding/xml"
	"os"
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

		if config.Keys[0] != "LogLevel" || config.Keys[1] != "Theme" {
			t.Fatalf("Expected key order ['LogLevel', 'Theme'], got %v", config.Keys)
		}
	})

	t.Run("Invalid XML", func(t *testing.T) {
		xmlData := `<Config><LogLevel>info<LogLevel></Config>` // malformed XML
		var config Config
		err := xml.Unmarshal([]byte(xmlData), &config)
		if err == nil {
			t.Fatal("Expected error, but got none")
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

		config, err := readAndParseXML(file.Name())
		if err != nil {
			t.Fatalf("Unexpected error reading XML: %v", err)
		}

		if config.Properties["LogLevel"] != "info" || config.Properties["Theme"] != "dark" {
			t.Fatalf("Expected properties not set correctly: %v", config.Properties)
		}

		if config.Keys[0] != "LogLevel" || config.Keys[1] != "Theme" {
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

		changed := updateConfigWithEnv(envVars, config, "CONFIGARR__")
		if len(changed) != 2 || changed["LogLevel"] != "debug" || changed["Theme"] != "light" {
			t.Fatalf("Expected changes not applied correctly: %v", changed)
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

		changed := updateConfigWithEnv(envVars, config, "CONFIGARR__")
		if len(changed) != 0 {
			t.Fatalf("Expected no changes, but got: %v", changed)
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

		envVars := []string{
			"CONFIGARR__LEVEL=LogLevel=debug",
		}

		// Prepare arguments to simulate command-line input
		args := []string{"cmd", "--config", file.Name(), "--prefix", "CONFIGARR__"}

		err = run(envVars, args)
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
	})
}
