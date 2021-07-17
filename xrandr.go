package main

import (
	"bufio"
	"fmt"
	"log"
	"os/exec"
	"regexp"
	"strings"
)

const (
	// PARSE_FIND_NEXT_CONNECTED_DISPLAY indicates that the parser should
	// find the next display in the connected state.
	PARSE_FIND_NEXT_CONNECTED_DISPLAY = iota
	// PARSE_FIND_EDID indicates that we've found a connected display and
	// that the parser should find the display's EDID.
	PARSE_FIND_EDID = iota
	// PARSE_EDID indicates that we've found the EDID and should now parse
	// it.
	PARSE_EDID = iota
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

	dispRE := regexp.MustCompile(`^(\S+) (disconnected|connected)`)
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

func noteConnectedDisplay(xrandrName string, serial uint32, isLaptop bool) {
	for i := range Config.Displays {
		display := &Config.Displays[i]
		if display.Serial == serial && display.IsLaptop == isLaptop {
			display.xrandrName = xrandrName
			display.connected = true
			log.Printf("display %q (xrandrName %s) is connected\n",
				display.Name, display.xrandrName)
			return
		}
	}
	log.Printf("display %q (serial %d, isLaptop %v) not found in config\n",
		xrandrName, serial, isLaptop)
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

	// Explicitly disable all unused displays. Without this, displays from
	// previous Layouts will remain on, whether they're connected or not.
	for xrandrName := range allDisplays {
		cmd = append(cmd, "--output", xrandrName, "--off")
	}
	return cmd
}
