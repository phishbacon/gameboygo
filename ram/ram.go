package ram

type WRAM [0x2000]uint8
type HRAM [0x007F]uint8

func (r *RAM) Read(address uint16) uint8 {
	if address >= 0xC000 && address <= 0xDFFF {
		return r.wram[address-0xC000]
	} else {
		return r.hram[address-0xFF80]
	}
}

func (r *RAM) Write(address uint16, value uint8) {
	if address >= 0xC000 && address <= 0xDFFF {
		r.wram[address-0xC000] = value
	} else {
		r.hram[address-0xFF80] = value
	}
}

type RAM struct {
	wram *WRAM
	hram *HRAM
}

func NewRam() *RAM {
	var wram WRAM
	var hram HRAM
	return &RAM{
		wram: &wram,
		hram: &hram,
	}
}
