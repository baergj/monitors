package main

import (
	"bufio"
	"flag"
	"io"
	"log"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

var configPath string
var pretend bool
var xdisplay string

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

func chooseLayout() Layout {
	for _, layout := range Config.Layouts {
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
	for _, display := range Config.Displays {
		if display.Name == name && display.connected {
			return true
		}
	}
	return false
}
