package ram

type WRAM [0x2000]uint8
type HRAM [0x007F]uint8

func (w *WRAM) Read(address uint16) uint8 {
	return w[address-0xC000]
}

func (h *HRAM) Read(address uint16) uint8 {
	return h[address-0xFF80]
}

func (w *WRAM) Write(address uint16, value uint8) {
	w[address-0xC000] = value
}

func (h *HRAM) Write(address uint16, value uint8) {
	h[address-0xFF80] = value
}

type RAM struct {
	WRAM *WRAM
	HRAM *HRAM
}

func NewRam() *RAM {
	var wram WRAM
	var hram HRAM
	return &RAM{
		WRAM: &wram,
		HRAM: &hram,
	}
}
