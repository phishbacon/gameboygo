package bus

import (
	"goboy/cart"
	"goboy/util"
)

type Bus struct {
	cart *cart.Cart
  HRAM *[0x7F]uint8
}

// Connect cart to the bus
func (b *Bus) ConnectCart(cartData *[]byte) {
	b.cart = (*cart.Cart)(cartData)
}

func (b *Bus) InitHRAM() {
  b.HRAM = &[0x7f]uint8{}
}

func (b *Bus) Read(address uint16) uint8 {
	if address < 0x8000 {
		if b.cart != nil {
			return b.cart.Read(address)
		} else {
			return util.NilRegister(address)
		}
	}

	return util.NotImplemented()
}

func (b *Bus) Write(address uint16, value uint8) {
  util.NilRegister(address)
}
