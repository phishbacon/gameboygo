package dbg

import (
	"fmt"
	"os"
)

var msg [0x400]uint8
var msgIndex uint8 = 0

func Update(Read func(uint16) uint8, Write func(uint16, uint8)) {
	if val := Read(0xFF02); val == 0x0081 {
		msg[msgIndex] = Read(0xFF01)
		if msg[msgIndex] > 127 {
			os.Exit(-1)
		}
		msgIndex++
		Write(0xFF02, 0x0000)
	}
}
func Print() bool {
	if msg[0] != 0 {
		fmt.Printf("Debug: %s\n", msg)
		return true
	}
	return false
}
