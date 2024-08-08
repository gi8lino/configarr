package main

import (
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/pflag"
)

// Config represents the XML structure with properties as a map and key order tracking.
type Config struct {
	XMLName    xml.Name          `xml:"Config"`
	Properties map[string]string `xml:"-"`
	Keys       []string          `xml:"-"`
}

// Flags represents the command-line flags used by the application.
type Flags struct {
	ConfigFilePath string
	Prefix         string
	Silent         bool
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
			return err
		}

		switch t := token.(type) {
		case xml.StartElement:
			var content string
			if err := d.DecodeElement(&content, &t); err != nil {
				return err
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
		return err
	}

	// Marshal in the order stored in Keys
	for _, key := range c.Keys {
		value := c.Properties[key]
		elem := xml.StartElement{Name: xml.Name{Local: key}}
		if err := e.EncodeElement(value, elem); err != nil {
			return err
		}
	}

	return e.EncodeToken(xml.EndElement{Name: start.Name})
}

// readAndParseXML reads and parses the XML file into a Config struct.
func readAndParseXML(xmlFile string) (*Config, error) {
	if _, err := os.Stat(xmlFile); err != nil {
		return nil, err // Return error if the file does not exist
	}

	file, err := os.ReadFile(xmlFile)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := xml.Unmarshal(file, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// updateConfigWithEnv updates the Config map with values from environment variables
// that match the given prefix. Returns a map of changed properties.
func updateConfigWithEnv(environ []string, config *Config, prefix string) map[string]string {
	changedProperties := make(map[string]string)
	envPrefix := strings.ToUpper(prefix)

	for _, envVar := range environ {
		// Split the environment variable into key and value
		if strings.HasPrefix(envVar, envPrefix) {
			parts := strings.SplitN(envVar[len(envPrefix):], "=", 2)
			if len(parts) != 2 {
				continue
			}

			// Extract the property key and its value from the environment variable
			envKeyValue := strings.SplitN(parts[1], "=", 2)
			if len(envKeyValue) != 2 {
				continue
			}

			envKey := envKeyValue[0]
			envValue := envKeyValue[1]

			// Update the config if the environment variable is different
			if currentValue, exists := config.Properties[envKey]; exists && envValue != currentValue {
				config.Properties[envKey] = envValue
				changedProperties[envKey] = envValue
			}
		}
	}

	return changedProperties
}

// writeConfigToFile writes the updated Config map back to the XML file.
func writeConfigToFile(config *Config, xmlFile string) error {
	output, err := xml.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	if err := os.WriteFile(xmlFile, output, 0644); err != nil {
		return err
	}

	return nil
}

// parseFlags parses the provided command-line flags and returns a Flags struct.
func parseFlags(flags []string) (Flags, error) {
	// Create a new flag set to avoid affecting the global command line flags
	flagSet := pflag.NewFlagSet("configFlags", pflag.ContinueOnError)

	configFilePath := flagSet.String("config", "/config/config.xml", "Path to the XML configuration file")
	prefix := flagSet.String("prefix", "CONFIGARR__", "Prefix for environment variables")
	silent := flagSet.Bool("silent", false, "Suppress output")

	if err := flagSet.Parse(flags); err != nil {
		return Flags{}, fmt.Errorf("error parsing flags: %w", err)
	}

	return Flags{
		ConfigFilePath: *configFilePath,
		Prefix:         *prefix,
		Silent:         *silent,
	}, nil
}

// run performs the main logic of the application, handling XML configuration updates.
func run(environ []string, args []string) error {
	flags, err := parseFlags(args[1:]) // exclude the program name
	if err != nil {
		return fmt.Errorf("error parsing flags: %w", err)
	}

	config, err := readAndParseXML(flags.ConfigFilePath)
	if err != nil {
		return fmt.Errorf("error reading XML file: %w", err)
	}

	changedProperties := updateConfigWithEnv(environ, config, flags.Prefix)
	if len(changedProperties) > 0 {
		if !flags.Silent {
			for key, value := range changedProperties {
				fmt.Printf("Updated '%s' to '%s'\n", key, value)
			}
		}

		if err := writeConfigToFile(config, flags.ConfigFilePath); err != nil {
			return fmt.Errorf("error writing updated configuration to XML file: %w", err)
		}
		return nil
	}

	if !flags.Silent {
		fmt.Println("No updates made to the configuration.")
	}

	return nil
}

func main() {
	if err := run(os.Environ(), os.Args); err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}
}
