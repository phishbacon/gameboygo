package cpu

import "goboy/bus"
import "goboy/cpu/registers"

type CPU struct {
  Registers registers.Registers
  Bus *bus.Bus
}

func NewCPU() CPU {
  bus := &bus.Bus{}
	cpu := CPU{
    Bus: bus,
  }
	return cpu
}
