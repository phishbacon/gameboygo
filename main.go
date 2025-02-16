package main

import "fmt"
// import "goboy/memory"
import "goboy/cpu"


func main() {
  // RAM := memory.NewMemory()
  CPU := cpu.NewCPU()
  CPU.Registers.F = 0xF0
  CPU.Registers.UnsetFlag(cpu.CARRY_FLAG)
  CPU.Registers.UnsetFlag(cpu.SUBTRACTION_FLAG)
  test := CPU.Registers.GetFlag(cpu.HALF_CARRY_FLAG)
  fmt.Println(test)
  test = CPU.Registers.GetFlag(cpu.CARRY_FLAG)
  fmt.Println(test)
  fmt.Printf("%08b\n", CPU.Registers.F)
}

