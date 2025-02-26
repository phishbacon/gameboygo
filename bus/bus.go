package bus

import (
	"goboy/apu"
	"goboy/cart"
	"goboy/cpu"
	"goboy/ppu"
	"goboy/ram"
	"goboy/timer"
	"goboy/util"
)

type Bus struct {
	cpu   *cpu.CPU
	apu   *apu.APU
	ppu   *ppu.PPU
	cart  *cart.Cart
	ram   *ram.RAM
	timer *timer.Timer
}

func NewBus(cpu *cpu.CPU, apu *apu.APU, ppu *ppu.PPU, timer *timer.Timer) *Bus {
	return &Bus{
		cpu:   cpu,
		apu:   apu,
		ppu:   ppu,
		ram:   ram.NewRam(),
		timer: timer,
	}
}

// Connect cart to the bus
func (b *Bus) ConnectCart(cartData *[]byte) {
	b.cart = (*cart.Cart)(cartData)
}

func (b *Bus) Read(address uint16) uint8 {
	if address < 0x8000 {
		if b.cart != nil {
			return b.cart.Read(address)
		} else {
			return util.NilRegister(address)
		}
	} else if address < 0xA000 {
		// 8 KiB VRAM
		return util.NilRegister(address)
	} else if address < 0xE000 {
		// 8 KiB WRAM
		if b.ram != nil && b.ram.WRAM != nil {
			return b.ram.WRAM.Read(address)
		} else {
			return util.NilRegister(address)
		}
	} else if address < 0xFE00 {
		// Unused Echo RAM
		return util.NilRegister(address)
	} else if address < 0xFEA0 {
		// Object attribute memory
		return util.NilRegister(address)
	} else if address < 0xFF00 {
		// Not usable
		return util.NilRegister(address)
	} else if address < 0xFF80 {
		// I/O Registers
		if address <= 0xFF07 && address >= 0xFF04 {
			return b.timer.Read(address)
		}
	} else if address < 0xFFFF {
		// HRAM
		if b.ram != nil && b.ram.HRAM != nil {
			return b.ram.HRAM.Read(address)
		}
	} else if address == 0xFFFF {
		return 1
	}

	return util.NotImplemented()
}

func (b *Bus) Write(address uint16, value uint8) {
	if address < 0x8000 {
		if b.cart != nil {
			b.cart.Write(address, value)
		} else {
			util.NilRegister(address)
		}
	} else if address < 0xA000 {
		// 8 KiB VRAM
		util.NilRegister(address)
	} else if address < 0xE000 {
		// 8 KiB WRAM
		if b.ram != nil && b.ram.WRAM != nil {
			b.ram.WRAM.Write(address, value)
		} else {
			util.NilRegister(address)
		}
	} else if address < 0xFE00 {
		// Unused Echo RAM
		util.NilRegister(address)
	} else if address < 0xFEA0 {
		// Object attribute memory
		util.NilRegister(address)
	} else if address < 0xFF00 {
		// Not usable
		util.NilRegister(address)
	} else if address < 0xFF80 {
		// I/O Registers
		if address <= 0xFF07 && address >= 0xFF04 {
			b.timer.Write(address, value)
		} else {
			util.NilRegister(address)
		}
	} else if address < 0xFFFF {
		// HRAM
		if b.ram != nil && b.ram.HRAM != nil {
			b.ram.HRAM.Write(address, value)
		}
	} else if address == 0xFFFF {
		return
	}
}
