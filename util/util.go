package util

import (
	"fmt"
	"os"
)

func NotImplemented() uint8 {
	fmt.Println("Not implemented!!")
	os.Exit(-1)
	return 1
}

func NilRegister(address uint16) uint8 {
	fmt.Printf("Can't read/write to %04x\n", address)
	os.Exit(-1)
	return 1
}

func If[T any](cond bool, vtrue, vfalse T) T {
	if cond {
		return vtrue
	}
	return vfalse
}
