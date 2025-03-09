package bus

import (
	"fmt"

	"github.com/phishbacon/gameboygo/cart"
	"github.com/phishbacon/gameboygo/common"
	"github.com/phishbacon/gameboygo/io"
	"github.com/phishbacon/gameboygo/ram"
)

type Bus struct {
	cart *cart.Cart
	ram  *ram.RAM
	io   *io.IO
	IE   uint8
}

func NewBus() *Bus {
	return &Bus{
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
			return common.ReadNilRegister(address)
		}
	} else if address < 0xA000 {
		// 8 KiB VRAM
		return common.ReadNilRegister(address)
	} else if address < 0xE000 {
		// 8 KiB WRAM
		if address <= 0xBFFF {
			return common.ReadNilRegister(address)
		}
		return b.ram.Read(address)
	} else if address < 0xFE00 {
		// Unused Echo RAM
		return common.ReadNilRegister(address)
	} else if address < 0xFEA0 {
		// Object attribute memory
		return common.ReadNilRegister(address)
	} else if address < 0xFF00 {
		// Not usable
		return common.ReadNilRegister(address)
	} else if address < 0xFF80 {
		// I/O Registers
		if address >= 0xFF00 && address <= 0xFF7F {
			return b.io.Read(address)
		}
		return common.ReadNilRegister(address)
	} else if address < 0xFFFF {
		// HRAM
		return b.ram.Read(address)
	} else if address == 0xFFFF {
		return b.IE
	}
	fmt.Printf("%04x\n", address)	
	panic("Trying to read non existent memory")
}

func (b *Bus) Write(address uint16, value uint8) {
	if address < 0x8000 {
		if b.cart != nil {
			b.cart.Write(address, value)
		} else {
			common.WriteNilRegister(address)
		}
	} else if address < 0xA000 {
		// 8 KiB VRAM
		common.WriteNilRegister(address)
	} else if address < 0xE000 {
		// 8 KiB WRAM
		b.ram.Write(address, value)
	} else if address < 0xFE00 {
		// Unused Echo RAM
		common.WriteNilRegister(address)
	} else if address < 0xFEA0 {
		// Object attribute memory
		common.WriteNilRegister(address)
	} else if address < 0xFF00 {
		// Not usable
		common.WriteNilRegister(address)
	} else if address < 0xFF80 {
		// I/O Registers
		if address >= 0xFF00 && address <= 0xFF7F {
			b.io.Write(address, value)
		} else {
			common.WriteNilRegister(address)
		}
	} else if address < 0xFFFF {
		// HRAM
		b.ram.Write(address, value)
	} else if address == 0xFFFF {
		b.IE = value
	}
}

func (b *Bus) DumpMemory() [0x10000]uint8 {
	memory := [0x10000]uint8{}
	for i := 0; i < len(memory); i++ {
		memory[i] = b.Read(uint16(i))
	}
	// var i uint16 = 0x0000
	// for i <= 0xFFFF {
	// 	memory[i] = b.Read(i)
	// 	i++
	// }
	return memory
}
