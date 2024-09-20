package main

import (
	"encoding/json"
	"fmt"
	"os"
)

type Config struct {
	Services []Service `json:"services"`
}

func LoadConfigOrCreate(filePath string) (*Config, error) {
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		defaultConfig := &Config{
			Services: []Service{},
		}
		file, err := os.Create(filePath)
		if err != nil {
			return nil, fmt.Errorf("failed to create config file: %v", err)
		}
		defer file.Close()

		encoder := json.NewEncoder(file)
		encoder.SetIndent("", "    ")
		if err := encoder.Encode(defaultConfig); err != nil {
			return nil, fmt.Errorf("failed to write default config: %v", err)
		}
		fmt.Println("Created default config file:", filePath)
		return defaultConfig, nil
	}

	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open config file: %v", err)
	}
	defer file.Close()

	var config Config
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&config); err != nil {
		return nil, fmt.Errorf("failed to decode config file: %v", err)
	}

	return &config, nil
}

func SaveConfig(filename string, config *Config) error {
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filename, data, 0644)
}
