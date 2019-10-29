package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"os/user"
	"path"
	"regexp"
	"strings"
)

var configPath string
var pretend bool
var xdisplay string

type Display struct {
	Name       string `json:"name"`
	Serial     string `json:"serial,omitempty"`
	IsLaptop   bool   `json:"is-laptop,omitempty"`
	connected  bool
	xrandrName string
}

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

var config struct {
	Displays []Display `json:"displays"`
	Layouts  []Layout
}

var displayByName map[string]*Display

func main() {
	parseOptions()
	parseConfig()

	if err := os.Setenv("DISPLAY", xdisplay); err != nil {
		log.Fatal(err)
	}

	allDisplays := detectDisplays()
	layout := chooseLayout()
	args := composeXrandrArgs(allDisplays, layout)
	log.Printf("xrandr cmd: /usr/bin/xrandr %s", strings.Join(args, " "))
	if pretend {
		log.Print("--pretend given; exiting")
		return
	}
	log.Print("applying layout...")
	xrandr := exec.Command("/usr/bin/xrandr", args...)
	if err := xrandr.Run(); err != nil {
		log.Fatal(err)
	}
}

func parseOptions() {
	flag.StringVar(&configPath, "config-path", ".config/monitors/config.json", "path to the config file")
	flag.BoolVar(&pretend, "pretend", false, "print what would have been executed and exit")
	flag.StringVar(&xdisplay, "xdisplay", ":0", "which X display to manage")
	flag.Parse()
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
	err = decoder.Decode(&config)
	if err != nil {
		log.Fatal(fmt.Sprintf("failed to parse config: %v", err))
	}

	displayByName = make(map[string]*Display)
	for i := range config.Displays {
		display := &config.Displays[i]
		displayByName[display.Name] = display
	}
}

const (
	PARSE_FIND_NEXT_CONNECTED_DISPLAY = iota
	PARSE_FIND_EDID                   = iota
	PARSE_EDID                        = iota
)

func detectDisplays() map[string]struct{} {
	xrandr := exec.Command("/usr/bin/xrandr", "--properties")
	xrandrOut, err := xrandr.StdoutPipe()
	if err != nil {
		log.Fatal(err)
	}
	if err := xrandr.Start(); err != nil {
		log.Fatal(err)
	}

	dispRE := regexp.MustCompile(`^(\w+) (disconnected|connected)`)
	edidStartRE := regexp.MustCompile(`^\s+EDID:`)
	edidRE := regexp.MustCompile(`^\s+[a-f0-9]+`)

	allDisplays := make(map[string]struct{})
	parseState := PARSE_FIND_NEXT_CONNECTED_DISPLAY
	var xrandrName, edid string
	var isLaptop bool
	sc := bufio.NewScanner(xrandrOut)
	for sc.Scan() {
		line := sc.Text()
		switch parseState {
		case PARSE_FIND_NEXT_CONNECTED_DISPLAY:
			parts := dispRE.FindStringSubmatch(line)
			if len(parts) == 0 {
				continue
			}
			xrandrName = parts[1]
			allDisplays[xrandrName] = struct{}{}
			if parts[2] != "connected" {
				continue
			}
			// Laptop displays may not have serial numbers, so we
			// need some other way to identify them.
			isLaptop = strings.HasPrefix(xrandrName, "eDP")
			parseState = PARSE_FIND_EDID
		case PARSE_FIND_EDID:
			if !edidStartRE.MatchString(line) {
				continue
			}
			parseState = PARSE_EDID
		case PARSE_EDID:
			if edidRE.MatchString(line) {
				edid += strings.TrimSpace(line)
				continue
			}
			serial := decodeSerial(edid)
			noteConnectedDisplay(xrandrName, serial, isLaptop)
			edid = ""
			parseState = PARSE_FIND_NEXT_CONNECTED_DISPLAY
		}
	}
	return allDisplays
}

func decodeSerial(edid string) string {
	cmd := exec.Command("edid-decode")
	in, err := cmd.StdinPipe()
	if err != nil {
		log.Fatal(err)
	}
	out, err := cmd.StdoutPipe()
	if err != nil {
		log.Fatal(err)
	}

	go func() {
		io.WriteString(in, edid)
		in.Close()
	}()

	re := regexp.MustCompile(`Serial Number (\w+)`)
	sc := bufio.NewScanner(out)
	cmd.Start()
	var serial string
	for sc.Scan() {
		parts := re.FindStringSubmatch(sc.Text())
		if len(parts) > 0 {
			serial = parts[1]
		}
	}

	return serial
}

func noteConnectedDisplay(xrandrName, serial string, isLaptop bool) {
	for i := range config.Displays {
		display := &config.Displays[i]
		if display.Serial == serial && display.IsLaptop == isLaptop {
			display.xrandrName = xrandrName
			display.connected = true
			log.Printf("display %q (xrandrName %s) is connected\n",
				display.Name, display.xrandrName)
			return
		}
	}
	log.Printf("display %q (serial %s, isLaptop %v) not found in config\n",
		xrandrName, serial, isLaptop)
}

func chooseLayout() Layout {
	for _, layout := range config.Layouts {
		match := true
		for _, display := range layout.Displays {
			if !isConnected(display.Display) {
				log.Printf("layout %q excluded - display %q not connected\n",
					layout.Name, display.Display)
				match = false
				break
			}
		}
		if match {
			log.Printf("layout %q matches connected displays\n", layout.Name)
			return layout
		}
	}
	log.Fatal("no layouts matched connected set of displays")
	return Layout{}
}

func isConnected(name string) bool {
	for _, display := range config.Displays {
		if display.Name == name && display.connected {
			return true
		}
	}
	return false
}

func composeXrandrArgs(allDisplays map[string]struct{}, layout Layout) []string {
	cmd := []string{}
	for _, display := range layout.Displays {
		xrandrName := displayByName[display.Display].xrandrName
		cmd = append(cmd, "--output", xrandrName, "--auto")
		if display.Primary {
			cmd = append(cmd, "--primary")
		}
		for _, pos := range display.Positions {
			cmd = append(cmd, fmt.Sprintf("--%s", pos.Position), displayByName[pos.Display].xrandrName)
		}

		delete(allDisplays, xrandrName)
	}

	// Explicitly disable all unused displays.
	for xrandrName, _ := range allDisplays {
		cmd = append(cmd, "--output", xrandrName, "--off")
	}
	return cmd
}
