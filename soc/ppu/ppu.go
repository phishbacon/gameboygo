package ppu

import "goboy/bus"

type PPU struct {
	Bus *bus.Bus
}

func NewPPU(bus *bus.Bus) PPU {
	ppu := PPU{
		Bus: bus,
	}
	return ppu
}
