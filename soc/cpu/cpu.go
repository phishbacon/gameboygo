package cpu

import "goboy/bus"
import "goboy/soc/cpu/registers"

type CPU struct {
	Registers registers.Registers
	Bus       *bus.Bus
}

func NewCPU(bus *bus.Bus) CPU {
	cpu := CPU{
		Bus: bus,
	}
	return cpu
}
