package main

import (
	"fmt"
	// import "goboy/memory"
	"goboy/cpu"
	"goboy/cpu/registers"
)

func main() {
  // RAM := memory.NewMemory()
  CPU := &cpu.CPU{}
  CPU.Registers.F = 0
  CPU.Registers.SetFlag(registers.ZERO_FLAG, true)
  CPU.Registers.SetFlag(registers.ZERO_FLAG, false)
  fmt.Printf("%08b\n", CPU.Registers.F)
}

