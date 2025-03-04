package soc

import (
	"fmt"
	"goboy/apu"
	"goboy/bus"
	"goboy/cpu"
	"goboy/dbg"
	"goboy/display"
	"goboy/ppu"
	"goboy/timer"

	qt "github.com/mappu/miqt/qt6"
)

type ComponentEnum uint8

const (
	APU ComponentEnum = 0
	CPU ComponentEnum = 1
	PPU ComponentEnum = 2
)

type SOC struct {
	APU        *apu.APU
	CPU        *cpu.CPU
	PPU        *ppu.PPU
	Timer      *timer.Timer
	Bus        *bus.Bus
	Running    bool
	Paused     bool
	Ticks      uint64
	TotalSteps uint64
	Display    *display.Display
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
		s.Step(1)
		s.TotalSteps++
		s.Ticks++
	}
}

func (s *SOC) Step(steps int) {
	for i := 0; i < steps; i++ {
		fmt.Printf("%x: ", s.TotalSteps)
		s.CPU.Step()
		UpdateTextEvent := display.NewUpdateTextEvent()
		qt.QCoreApplication_PostEvent(s.Display.InstrText.QObject, UpdateTextEvent)
		s.TotalSteps++
		dbg.Update(s.Bus.Read, s.Bus.Write)
		dbg.Print()
		// s.APU.Step()
		// s.PPU.Step()
	}
}
