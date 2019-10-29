package main

import (
	"bufio"
	"io"
	"log"
	"os/exec"
	"regexp"
)

// decodeSerial obtains the serial number from the given EDID block.
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
