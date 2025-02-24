package soc

import (
	"goboy/apu"
	"goboy/bus"
	"goboy/cpu"
	"goboy/ppu"
	"goboy/util"
)

type ComponentEnum uint8

const (
	APU ComponentEnum = 0
	CPU ComponentEnum = 1
	PPU ComponentEnum = 2
)

type SOC struct {
	APU apu.APU
	CPU cpu.CPU
	PPU ppu.PPU
}

func NewSOC(bus *bus.Bus) *SOC {
	return &SOC{
		APU: *apu.NewAPU(bus),
		CPU: *cpu.NewCPU(bus),
		PPU: *ppu.NewPPU(bus),
	}
}

func (s *SOC) Init() {
  s.CPU.Init()
  // s.APU.Init()
  // s.PPU.Init()
}

func (s *SOC) Step() {
  s.CPU.Step()
  // s.APU.Step()
  // s.PPU.Step()
}

// only for the cpu right now
func (s *SOC) Read(address uint16, component ComponentEnum) uint8 {
	switch component {
	case APU:
		return util.NotImplemented()
	case CPU:
		return s.CPU.Read(address)
	case PPU:
		return util.NotImplemented()
	default:
		return util.NotImplemented()
	}
}

func (s *SOC) Write(address uint16, value uint8, component ComponentEnum) {
	switch component {
	case APU:
		util.NotImplemented()
	case CPU:
		s.CPU.Write(address, value)
	case PPU:
		util.NotImplemented()
	default:
		util.NotImplemented()
	}
}
