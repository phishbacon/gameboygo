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
	APU     *apu.APU
	CPU     *cpu.CPU
	PPU     *ppu.PPU
	Timer   *timer.Timer
	Bus     *bus.Bus
	running bool
	paused  bool
	ticks   uint64
}

func NewSOC() *SOC {
	apu := apu.NewAPU()
	cpu := cpu.NewCPU()
	ppu := ppu.NewPPU()
	bus := bus.NewBus(cpu, apu, ppu)
	cpu.SetReadWrite(bus.Read, bus.Write)
	return &SOC{
		APU: apu,
		CPU: cpu,
		PPU: ppu,
		Bus: bus,
	}
}

func (s *SOC) Init() {
	s.CPU.Init()
	// s.APU.Init()
	// s.PPU.Init()
	s.running = true
	s.paused = false
	s.ticks = 0

	for s.running {
		if s.paused {
			continue
		}
		s.Step()
		s.paused = true
		s.ticks++
	}
}

func (s *SOC) Step() {
	s.CPU.Step()
	// s.APU.Step()
	// s.PPU.Step()
}
