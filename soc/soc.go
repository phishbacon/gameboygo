package soc

import (
	"goboy/apu"
	"goboy/bus"
	"goboy/cpu"
	"goboy/ppu"
	"goboy/timer"
)

type ComponentEnum uint8

const (
	APU ComponentEnum = 0
	CPU ComponentEnum = 1
	PPU ComponentEnum = 2
)

type SOC struct {
	APU   *apu.APU
	CPU   *cpu.CPU
	PPU   *ppu.PPU
	Timer *timer.Timer
	Bus   *bus.Bus
}

func NewSOC() *SOC {
	apu := apu.NewAPU()
	cpu := cpu.NewCPU()
	ppu := ppu.NewPPU()
	timer := new(timer.Timer)
	bus := bus.NewBus(cpu, apu, ppu, timer)
	cpu.SetReadWrite(bus.Read, bus.Write)
	return &SOC{
		APU:   apu,
		CPU:   cpu,
		PPU:   ppu,
		Timer: timer,
		Bus:   bus,
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
