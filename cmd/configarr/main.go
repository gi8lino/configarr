package main

import (
	"encoding/xml"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"

	"github.com/spf13/pflag"
)

// Constants for default configuration
const (
	DefaultConfigPath = "/config/config.xml"
	DefaultPrefix     = "CONFIGARR__"
)

// Config represents the XML structure with properties as a map and key order tracking.
type Config struct {
	XMLName    xml.Name          `xml:"Config"`
	Properties map[string]string `xml:"-"`
	Keys       []string          `xml:"-"`
}

// Flags represents the command-line flags used by the application.
type Flags struct {
	ConfigFilePath      string
	IgnoreMissingConfig bool
	Prefix              string
	Debug               bool
}

// UnmarshalXML customizes the unmarshalling of the XML into the Config struct.
// This function reads XML elements and stores them in the Properties map and tracks key order.
func (c *Config) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	c.Properties = make(map[string]string)
	c.Keys = []string{}
	for {
		token, err := d.Token()
		if err != nil {
			if err == io.EOF {
				break // End of XML document
			}
			return fmt.Errorf("error parsing XML token: %w", err)
		}

		switch t := token.(type) {
		case xml.StartElement:
			var content string
			if err := d.DecodeElement(&content, &t); err != nil {
				return fmt.Errorf("error decoding XML element %s: %w", t.Name.Local, err)
			}
			// Store the element's content in the map
			c.Properties[t.Name.Local] = content
			// Track the key order
			c.Keys = append(c.Keys, t.Name.Local)
		}
	}
	return nil
}

// MarshalXML customizes the marshalling of the Config struct into XML.
// It encodes the Properties map into XML elements preserving the key order.
func (c *Config) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	start.Name.Local = "Config"
	if err := e.EncodeToken(start); err != nil {
		return fmt.Errorf("error encoding XML start token: %w", err)
	}

	// Marshal in the order stored in Keys
	for _, key := range c.Keys {
		value := c.Properties[key]
		elem := xml.StartElement{Name: xml.Name{Local: key}}
		if err := e.EncodeElement(value, elem); err != nil {
			return fmt.Errorf("error encoding XML element %s: %w", key, err)
		}
	}

	if err := e.EncodeToken(xml.EndElement{Name: start.Name}); err != nil {
		return fmt.Errorf("error encoding XML end token: %w", err)
	}

	return nil
}

// readAndParseXML reads and parses the XML file into a Config struct.
func readAndParseXML(xmlFile string) (*Config, error) {
	if _, err := os.Stat(xmlFile); os.IsNotExist(err) {
		return nil, fmt.Errorf("file does not exist: %s", xmlFile)
	}

	file, err := os.ReadFile(xmlFile)
	if err != nil {
		return nil, fmt.Errorf("error reading file %s: %w", xmlFile, err)
	}

	var cfg Config
	if err := xml.Unmarshal(file, &cfg); err != nil {
		return nil, fmt.Errorf("error unmarshalling XML: %w", err)
	}

	return &cfg, nil
}

// updateConfigWithEnv updates the Config map with values from environment variables
// that match the given prefix. Returns a map of changed properties.
func updateConfigWithEnv(environ []string, config *Config, prefix string, logger *slog.Logger) map[string]string {
	changedProperties := make(map[string]string)
	envPrefix := strings.ToUpper(prefix)

	for _, envVar := range environ {
		if !strings.HasPrefix(envVar, envPrefix) { // Check if the environment variable starts with the prefix
			continue
		}

		// Split the environment variable into key and value
		parts := strings.SplitN(envVar[len(envPrefix):], "=", 2)
		if len(parts) != 2 {
			logger.Warn(fmt.Sprintf("Invalid environment variable format: %s", envVar))
			continue
		}

		// Extract the property key and its value from the environment variable
		envKeyValue := strings.SplitN(parts[1], "=", 2)
		if len(envKeyValue) != 2 {
			logger.Warn(fmt.Sprintf("Invalid key-value pair in environment variable: %s", envVar))
			continue
		}

		envKey := envKeyValue[0]
		envValue := envKeyValue[1]

		// Update the config if the environment variable is different
		if currentValue, exists := config.Properties[envKey]; exists && envValue != currentValue {
			config.Properties[envKey] = envValue
			changedProperties[envKey] = envValue
			logger.Debug(fmt.Sprintf("Updated '%s' to '%s'", envKey, envValue))
		}
	}

	if len(changedProperties) == 0 {
		logger.Debug("No updates made to the configuration.")
	}

	return changedProperties
}

// writeConfigToFile writes the updated Config map back to the XML file.
func writeConfigToFile(config *Config, xmlFile string) error {
	output, err := xml.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("error marshalling XML: %w", err)
	}

	if err := os.WriteFile(xmlFile, output, 0644); err != nil {
		return fmt.Errorf("error writing file %s: %w", xmlFile, err)
	}

	return nil
}

// parseFlags parses the provided command-line flags and returns a Flags struct.
func parseFlags(flags []string) (Flags, error) {
	flagSet := pflag.NewFlagSet("configFlags", pflag.ContinueOnError) // Create a new flag set to avoid affecting the global command line flags

	configFilePath := flagSet.String("config", DefaultConfigPath, "Path to the XML configuration file")
	prefix := flagSet.String("prefix", DefaultPrefix, "Prefix for environment variables")
	debug := flagSet.Bool("debug", false, "Enable debug logging")
	ignoreMissingConfig := flagSet.Bool("ignore-missing-config", false, "Ignore missing configuration file")

	if err := flagSet.Parse(flags); err != nil {
		return Flags{}, fmt.Errorf("error parsing flags: %w", err)
	}

	return Flags{
		ConfigFilePath:      *configFilePath,
		IgnoreMissingConfig: *ignoreMissingConfig,
		Prefix:              *prefix,
		Debug:               *debug,
	}, nil
}

// run performs the main logic of the application, handling XML configuration updates.
func run(environ []string, args []string, output io.Writer) error {
	flags, err := parseFlags(args[1:]) // exclude the program name
	if err != nil {
		return err
	}

	level := slog.LevelInfo
	if flags.Debug {
		level = slog.LevelDebug
	}
	logger := slog.New(slog.NewTextHandler(output, &slog.HandlerOptions{Level: level}))

	// Attempt to read and parse the XML configuration file
	config, err := readAndParseXML(flags.ConfigFilePath)
	if err != nil {
		if strings.Contains(err.Error(), "file does not exist") && flags.IgnoreMissingConfig {
			logger.Debug("No configuration file found. Skipping update.")
			return nil
		}
		return fmt.Errorf("error reading XML file: %w", err)
	}

	updateConfigWithEnv(environ, config, flags.Prefix, logger)

	if err := writeConfigToFile(config, flags.ConfigFilePath); err != nil {
		return fmt.Errorf("error writing updated configuration to XML file: %w", err)
	}

	return nil
}

func main() {
	if err := run(os.Environ(), os.Args, os.Stdout); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}
