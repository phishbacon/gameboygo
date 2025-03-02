package bus

import (
	"goboy/apu"
	"goboy/cart"
	"goboy/cpu"
	"goboy/io"
	"goboy/ppu"
	"goboy/ram"
	"goboy/util"
)

type Bus struct {
	cpu  *cpu.CPU
	apu  *apu.APU
	ppu  *ppu.PPU
	cart *cart.Cart
	ram  *ram.RAM
	io   *io.IO
	IE   uint8
}

func NewBus(cpu *cpu.CPU, apu *apu.APU, ppu *ppu.PPU) *Bus {
	return &Bus{
		cpu: cpu,
		apu: apu,
		ppu: ppu,
		ram: ram.NewRam(),
		io:  new(io.IO),
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
			util.ReadNilRegister(address)
			return util.Exit()
		}
	} else if address < 0xA000 {
		// 8 KiB VRAM
		util.ReadNilRegister(address)
		return util.Exit()
	} else if address < 0xE000 {
		// 8 KiB WRAM
		return b.ram.Read(address)
	} else if address < 0xFE00 {
		// Unused Echo RAM
		util.ReadNilRegister(address)
		return util.Exit()
	} else if address < 0xFEA0 {
		// Object attribute memory
		util.ReadNilRegister(address)
		return util.Exit()
	} else if address < 0xFF00 {
		// Not usable
		util.ReadNilRegister(address)
		return util.Exit()
	} else if address < 0xFF80 {
		// I/O Registers
		if address >= 0xFF00 && address <= 0xFF7F {
			return b.io.Read(address)
		}
		util.ReadNilRegister(address)
	} else if address < 0xFFFF {
		// HRAM
		return b.ram.Read(address)
	} else if address == 0xFFFF {
		return b.IE
	}

	util.ReadNilRegister(address)
	return util.Exit()
}

func (b *Bus) Write(address uint16, value uint8) {
	if address < 0x8000 {
		if b.cart != nil {
			b.cart.Write(address, value)
		} else {
			util.WriteNilRegister(address)
		}
	} else if address < 0xA000 {
		// 8 KiB VRAM
		util.WriteNilRegister(address)
	} else if address < 0xE000 {
		// 8 KiB WRAM
		b.ram.Write(address, value)
	} else if address < 0xFE00 {
		// Unused Echo RAM
		util.WriteNilRegister(address)
	} else if address < 0xFEA0 {
		// Object attribute memory
		util.WriteNilRegister(address)
	} else if address < 0xFF00 {
		// Not usable
		util.WriteNilRegister(address)
	} else if address < 0xFF80 {
		// I/O Registers
		if address >= 0xFF07 && address <= 0xFF04 {
			b.io.Write(address, value)
		} else {
			util.WriteNilRegister(address)
		}
	} else if address < 0xFFFF {
		// HRAM
		b.ram.Write(address, value)
	} else if address == 0xFFFF {
		b.IE = value
	}
}
