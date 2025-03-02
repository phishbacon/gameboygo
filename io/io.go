package io

import "goboy/timer"

const (
	VBLANK uint8 = 0b00000001
	LCD    uint8 = 0b00000010
	TIMER  uint8 = 0b00000100
	SERIAL uint8 = 0b00001000
	JOYPAD uint8 = 0b00010000
)

type IO struct {
	joypad uint8
	timer  timer.Timer
	IF     uint8
}

func (i *IO) Read(address uint16) uint8 {
	if address == 0xFF00 {
		return i.joypad
	}

	if address >= 0xFF04 && address <= 0xFF07 {
		return i.timer.Read(address)
	}

	if address == 0xFF0F {
		return i.IF
	}

	return 0
}

func (i *IO) Write(address uint16, value uint8) {
	if address == 0xFF00 {
		i.joypad = value
	} else if address >= 0xFF04 && address <= 0xFF07 {
		i.timer.Write(address, value)
	} else if address == 0xFF0F {
		i.IF = value
	}
}
