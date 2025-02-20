package apu

import "goboy/bus"

type APU struct {
	Bus *bus.Bus
}

func NewAPU(bus *bus.Bus) APU {
	apu := APU{
		Bus: bus,
	}
	return apu
}
