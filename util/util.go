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

func NilRegister(register string) uint8 {
	fmt.Printf("Can't read/write %s as it doesn't exist\n", register)
	os.Exit(-1)
	return 1
}

func If[T any](cond bool, vtrue, vfalse T) T {
	if cond {
		return vtrue
	}
	return vfalse
}
