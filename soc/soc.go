package soc

import (
	"goboy/bus"
  "goboy/soc/apu"
	"goboy/soc/cpu"
  "goboy/soc/ppu"
)

type SOC struct {
  cpu cpu.CPU
  apu apu.APU
  ppu ppu.PPU
}

func NewSOC() SOC {
  bus := bus.Bus{}
  soc := SOC{
    cpu: cpu.NewCPU(&bus),
    apu: apu.NewAPU(&bus),
    ppu: ppu.NewPPU(&bus),
  }

  return soc
}
