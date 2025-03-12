package cpu

import (
	"github.com/phishbacon/gameboygo/common"
)

type RegisterName uint8
type RegisterName16 uint8

const (
	A RegisterName = iota
	B
	C
	D
	E
	F
	H
	L
	IME
)

const (
	AF RegisterName16 = iota
	BC
	DE
	HL
	SP
	PC
)

type Register struct {
	value *uint8
	name  RegisterName
}

type Register16 struct {
	value [2]*uint8
	name  RegisterName16
}
type Registers struct {
	A   *Register
	B   *Register
	C   *Register
	D   *Register
	E   *Register
	F   *Register
	H   *Register
	L   *Register
	AF  *Register16
	BC  *Register16
	DE  *Register16
	HL  *Register16
	SP  *Register16
	PC  *Register16
	IME *Register
}

func NewRegisters() *Registers {
	var a, b, c, d, e, f, h, l, ime, s, p, program, counter uint8
	return &Registers{
		A:   &Register{&a, A},
		B:   &Register{&b, B},
		C:   &Register{&c, C},
		D:   &Register{&d, D},
		E:   &Register{&e, E},
		F:   &Register{&f, F},
		H:   &Register{&h, H},
		L:   &Register{&l, L},
		SP:  &Register16{[2]*uint8{&s, &p}, SP},
		PC:  &Register16{[2]*uint8{&program, &counter}, PC},
		IME: &Register{&ime, IME},
		AF:  &Register16{[2]*uint8{&a, &f}, AF},
		BC:  &Register16{[2]*uint8{&b, &c}, BC},
		DE:  &Register16{[2]*uint8{&d, &e}, DE},
		HL:  &Register16{[2]*uint8{&h, &l}, HL},
	}
}

func (r *Register16) Add(value uint16) {
	// storing big endian style
	hi := uint16(*r.value[0]) << 8
	lo := uint16(*r.value[1])
	regValue := hi | lo
	regValue += value
	*r.value[1] = uint8(regValue & 0x00FF)
	*r.value[0] = uint8(regValue>>8) & 0x00FF
}

func (r *Register) Add(value uint8) {
	*r.value += value
}

func (r *Register16) Sub(value uint16) {
	// storing big endian style
	hi := uint16(*r.value[0]) << 8
	lo := uint16(*r.value[1])
	regValue := hi | lo
	regValue -= value
	*r.value[1] = uint8(regValue & 0x00FF)
	*r.value[0] = uint8(regValue>>8) & 0x00FF
}

func (r *Register) Sub(value uint8) {
	*r.value -= value
}

func (r *Register16) Value() uint16 {
	return (uint16(*r.value[0]) << 8) | uint16(*r.value[1])
}

func (r *Register) Value() uint8 {
	return *r.value
}

func (r *Register16) Equals(value uint16) {
	// storing big endian style
	*r.value[1] = uint8(value & 0x00FF)
	*r.value[0] = uint8(value>>8) & 0x00FF
}

func (r *Register) Equals(value uint8) {
	if r.name == F {
		*r.value = value & 0x00F0
	} else {
		*r.value = value
	}
}

// Returns true if the flag is flipped, false otherwise
func (r *Registers) GetFlag(flag common.FlagMask) bool {
	if *r.F.value&uint8(flag) > 0 {
		return true
	} else {
		return false
	}
}

// Sets the given flag to true or false
func (r *Registers) SetFlag(flag common.FlagMask, value bool) {
	if value {
		*r.F.value = (*r.F.value | uint8(flag)) & 0x00F0
	} else {
		*r.F.value = (*r.F.value & ^uint8(flag)) & 0x00F0
	}
}
