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

type SOC struct {
	APU            *apu.APU
	CPU            *cpu.CPU
	PPU            *ppu.PPU
	TIMER          *timer.Timer
	Bus            *bus.Bus
	Running        bool
	Paused         bool
	Ticks          uint64
	TotalSteps     uint64
	CpuStateString string
}

func NewSOC() *SOC {
	bus := bus.NewBus()
	apu := apu.NewAPU(bus)
	cpu := cpu.NewCPU(bus)
	ppu := ppu.NewPPU(bus)
	return &SOC{
		APU: apu,
		CPU: cpu,
		PPU: ppu,
		Bus: bus,
	}
}

func (s *SOC) ConnectCart(cart *[]byte) {
	s.Bus.ConnectCart(cart)
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
		s.Step(1)
	}
}

func (s *SOC) Step(steps uint8) {
	var i uint8 = 0
	for i < steps {
		fmt.Printf("%x: ", s.TotalSteps)
		s.CPU.Step()
		// if s.CPU.CurOpCode == 0x39 || s.CPU.CurOpCode == 0xE8 || s.CPU.CurOpCode == 0xF8 {
		// 	s.Paused = true
		// 	return
		// }
		s.TotalSteps++
		dbg.Update(s.Bus.Read, s.Bus.Write)
		dbg.Print()
		// s.APU.Step()
		// s.PPU.Step()
		s.Ticks++
		i++
	}
}
