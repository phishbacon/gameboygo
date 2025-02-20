package util

import (
	"fmt"
	"os"
)

func NotImplemented() uint8 {
	fmt.Println("Not implemented")
	os.Exit(-1)
  return 1
}

func If[T any](cond bool, vtrue, vfalse T) T {
  if cond {
    return vtrue
  }
  return vfalse
}

func DumpHex(cart []byte) {
  fo, err := os.Create("dump.txt")
  if err != nil {
    fmt.Print(err)
  }

  defer fo.Close()

  for i:=0; i<len(cart) - 16; i+=16 {
    str := fmt.Sprintf("0x%04x: ", i)
    str += fmt.Sprintf("%02x %02x %02x %02x ", cart[i], cart[i+1], cart[i+2], cart[i+3])
    str += fmt.Sprintf("%02x %02x %02x %02x ", cart[i+4], cart[i+5], cart[i+6], cart[i+7])
    str += fmt.Sprintf("%02x %02x %02x %02x ", cart[i+8], cart[i+9], cart[i+10], cart[i+11])
    str += fmt.Sprintf("%02x %02x %02x %02x\n", cart[i+12], cart[i+13], cart[i+14], cart[i+15])
    fo.WriteString(str)
  }
}
