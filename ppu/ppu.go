package ppu

import "goboy/bus"

type PPU struct {
	bus *bus.Bus
}

func NewPPU(bus *bus.Bus) *PPU {
	return &PPU{
		bus: bus,
	}
}
