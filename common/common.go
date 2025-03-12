package common

import (
	"fmt"
	"os"
)

type FlagMask uint8

const (
	ZERO_FLAG        FlagMask = 0b10000000
	SUBTRACTION_FLAG FlagMask = 0b01000000
	HALF_CARRY_FLAG  FlagMask = 0b00100000
	CARRY_FLAG       FlagMask = 0b00010000
)

func NotImplemented() uint8 {
	fmt.Println("Not implemented!!")
	os.Exit(-1)
	return 1
}

func WriteNilRegister(address uint16) {
	fmt.Printf("Can't write to %04x\n", address)
	if address >= 0x8000 && address <= 0x9FFF {
		return
	}
	os.Exit(-1)
}

func ReadNilRegister(address uint16) uint8 {
	// fmt.Printf("Can't read from %04x\n", address)
	return 0
}

func Exit() uint8 {
	os.Exit(-1)
	return 1
}

func If[T any](cond bool, vtrue, vfalse T) T {
	if cond {
		return vtrue
	}
	return vfalse
}
