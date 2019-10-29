package main

import (
	"flag"
	"log"
	"os"
	"os/exec"
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
