package soc

import (
	"fmt"

	"github.com/phishbacon/gameboygo/apu"
	"github.com/phishbacon/gameboygo/bus"
	"github.com/phishbacon/gameboygo/cpu"
	"github.com/phishbacon/gameboygo/dbg"
	"github.com/phishbacon/gameboygo/ppu"
	"github.com/phishbacon/gameboygo/timer"
)

type ComponentEnum uint8

const (
	APU ComponentEnum = 0
	CPU ComponentEnum = 1
	PPU ComponentEnum = 2
)

type SOC struct {
	APU                     *apu.APU
	CPU                     *cpu.CPU
	PPU                     *ppu.PPU
	Timer                   *timer.Timer
	Bus                     *bus.Bus
	Running                 bool
	Paused                  bool
	Ticks                   uint64
	TotalSteps              uint64
	CpuStateString          string
	CpuStateReadyForReading bool
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
	// define soc display elements
	s.CPU.Init()
	// s.APU.Init()
	// s.PPU.Init()
	s.Running = true
	s.Paused = true
	s.Ticks = 0

	for s.Running {
		if s.Paused {
			continue
		}
		s.CpuStateReadyForReading = false
		s.CpuStateString = s.Step()
		fmt.Println(s.CpuStateString)
		s.CpuStateReadyForReading = true
		s.TotalSteps++
		s.Ticks++
	}
}

func (s *SOC) Step() string {
	fmt.Printf("%x: ", s.TotalSteps)
	cpuString := s.CPU.Step()
	s.TotalSteps++
	dbg.Update(s.Bus.Read, s.Bus.Write)
	dbg.Print()
	// s.APU.Step()
	// s.PPU.Step()
	return cpuString
}
