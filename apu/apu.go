package apu

import "goboy/bus"

type APU struct {
	bus *bus.Bus
}

func NewAPU(bus *bus.Bus) *APU {
	return &APU{
		bus: bus,
	}
}
