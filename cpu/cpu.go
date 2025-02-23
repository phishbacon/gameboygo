package cpu

import (
	"goboy/bus"
	"goboy/cpu/registers"
)

type CPU struct {
	bus       *bus.Bus
	Registers *registers.Registers

  fetched   uint8
  abs_addr  uint16
  rel_addr  uint8
  opcode    uint8
  cycles    uint8
}

func NewCPU(bus *bus.Bus) *CPU {
	registers := new(registers.Registers)
	return &CPU{
		bus:       bus,
		Registers: registers,
	}
}

func (c *CPU) Read(address uint16) uint8 {
	return c.bus.Read(address)
}

func (c *CPU) Write(address uint16, value uint8) {
	c.bus.Write(address, value)
}
