package main

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"log"
	"reflect"
)

const (
	serialOffset = 12
)

var (
	magic = []byte{0x00, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0x00}
)

// decodeSerial obtains the serial number from the given EDID block.
func decodeSerial(edid string) uint32 {
	edidBytes, err := hex.DecodeString(edid)
	if err != nil {
		log.Fatal(err)
	}

	if bytes.Compare(edidBytes[0:8], magic) != 0 {
		log.Fatal("given EDID is not magical enough")
	}

	if uintptr(len(edidBytes)) < serialOffset+reflect.TypeOf(uint32(0)).Size() {
		log.Fatal("given EDID too short to contain a serial number")
	}

	return binary.LittleEndian.Uint32(edidBytes[serialOffset:])
}
