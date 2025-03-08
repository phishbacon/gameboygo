package apu

import "github.com/phishbacon/gameboygo/bus"

type APU struct {
	bus *bus.Bus
}

func NewAPU(bus *bus.Bus) *APU {
	return &APU{
		bus: bus,
	}
}
