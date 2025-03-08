package ppu

import "github.com/phishbacon/gameboygo/bus"

type PPU struct {
	bus *bus.Bus
}

func NewPPU(bus *bus.Bus) *PPU {
	return &PPU{
		bus: bus,
	}
}
