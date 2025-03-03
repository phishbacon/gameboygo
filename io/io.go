package io

import (
	"goboy/timer"
)

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
	SB     uint8
	SC     uint8
}

func (i *IO) Read(address uint16) uint8 {
	if address == 0xFF00 {
		return i.joypad
	}

	if address == 0xFF01 {
		return i.SB
	}

	if address == 0xFF02 {
		return i.SC
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
	}
	if address == 0xFF01 {
		i.SB = value
	}
	if address == 0xFF02 {
		i.SC = value
	} else if address >= 0xFF04 && address <= 0xFF07 {
		i.timer.Write(address, value)
	} else if address == 0xFF0F {
		i.IF = value
	}
}
