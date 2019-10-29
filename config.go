package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/user"
	"path"
)

// Display represents a named display from the configuration, identified either
// by its serial number or as the local laptop display (which sometimes does
// not have a serial number).
type Display struct {
	Name       string `json:"name"`
	Serial     uint32 `json:"serial,omitempty"`
	IsLaptop   bool   `json:"is-laptop,omitempty"`
	connected  bool   // connected indicates whether this Display has been detected.
	xrandrName string // xrandrName matches this Display with xrandr.
}

// Layout represents a named layout that comprises one or more Displays with
// their positions indicated relative to each other.
type Layout struct {
	Name     string `json:"name"`
	Displays []struct {
		Display   string `json:"display"`
		Primary   bool   `json:"primary,omitempty"`
		Positions []struct {
			Position string `json:"location"`
			Display  string `json:"display"`
		} `json:"relative-locations,omitempty"`
	} `json:"displays"`
}

// Config represents an entire config file.
var Config struct {
	Displays []Display `json:"displays"`
	Layouts  []Layout  `json:"layouts"`
}

func parseConfig() {
	usr, err := user.Current()
	if err != nil {
		log.Fatal(fmt.Sprintf("failed to get current user: %v", err))
	}
	file, err := os.Open(path.Join(usr.HomeDir, configPath))
	if err != nil {
		log.Fatal(fmt.Sprintf("failed to open config: %v", err))
	}
	decoder := json.NewDecoder(file)
	decoder.DisallowUnknownFields()
	err = decoder.Decode(&Config)
	if err != nil {
		log.Fatal(fmt.Sprintf("failed to parse config: %v", err))
	}

	displayByName = make(map[string]*Display)
	for i := range Config.Displays {
		display := &Config.Displays[i]
		displayByName[display.Name] = display
	}
}
