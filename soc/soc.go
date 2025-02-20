package soc

import (
	"goboy/bus"
	"goboy/cart"
	"goboy/soc/apu"
	"goboy/soc/cpu"
	"goboy/soc/ppu"
	"goboy/util"
)

type SOC struct {
	cpu  cpu.CPU
	apu  apu.APU
	ppu  ppu.PPU
	cart *cart.Cart
}

func NewSOC(cart *cart.Cart) SOC {
	bus := bus.Bus{}
	soc := SOC{
		cpu:  cpu.NewCPU(&bus),
		apu:  apu.NewAPU(&bus),
		ppu:  ppu.NewPPU(&bus),
		cart: cart,
	}

	return soc
}

func (s SOC) BusRead(address uint16) uint8 {
	if address < 0x8000 {
		return s.cart.Read(address)
	}

	return util.NotImplemented()
}

func (s SOC) BusWrite(address uint16, value uint8) {
	if address < 0x8000 {
		s.cart.Write(address, value)
	}
}
