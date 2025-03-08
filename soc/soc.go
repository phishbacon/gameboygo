package soc

import (
	"fmt"

	"github.com/phishbacon/gameboygo/bus"
	"github.com/phishbacon/gameboygo/cpu"
	"github.com/phishbacon/gameboygo/apu"
	"github.com/phishbacon/gameboygo/dbg"
	"github.com/phishbacon/gameboygo/ppu"
	"github.com/phishbacon/gameboygo/timer"
)

type SOC struct {
	apu                     *apu.APU
	cpu                     *cpu.CPU
	ppu                     *ppu.PPU
	timer                   *timer.Timer
	bus                     *bus.Bus
	running                 bool
	Paused                  bool
	ticks                   uint64
	totalSteps              uint64
}

func NewSOC() *SOC {
	bus := bus.NewBus()
	apu := apu.NewAPU(bus)
	cpu := cpu.NewCPU(bus)
	ppu := ppu.NewPPU(bus)
	return &SOC{
		apu: apu,
		cpu: cpu,
		ppu: ppu,
		bus: bus,
	}
}

func (s *SOC) ConnectCart(cart *[]byte) {
	s.bus.ConnectCart(cart)
}

func (s *SOC) Init() {
	// define soc display elements
	s.cpu.Init()
	// s.APU.Init()
	// s.PPU.Init()
	s.running = true
	s.Paused = true
	s.ticks = 0

	for s.running {
		if s.Paused {
			continue
		}
		s.Step()
		s.totalSteps++
		s.ticks++
	}
}

func (s *SOC) Step() string {
	fmt.Printf("%x: ", s.totalSteps)
	cpuString := s.cpu.Step()
	s.totalSteps++
	dbg.Update(s.bus.Read, s.bus.Write)
	dbg.Print()
	// s.APU.Step()
	// s.PPU.Step()
	return cpuString
}
