package registers

import "fmt"

type FlagMask uint16

const (
	ZERO_FLAG        FlagMask = 0b10000000
	SUBTRACTION_FLAG FlagMask = 0b01000000
	HALF_CARRY_FLAG  FlagMask = 0b00100000
	CARRY_FLAG       FlagMask = 0b00010000
)

type Reg uint8

const (
	A Reg = iota
	F Reg = iota
	B Reg = iota
	C Reg = iota
	D Reg = iota
	E Reg = iota
	H Reg = iota
	L Reg = iota
)

type Registers struct {
	af  uint16
	bc  uint16
	de  uint16
	hl  uint16
	sp  uint16 // Stack Pointer
	pc  uint16 // Program Counter
	ime bool   // Interrupt master enable
}

func (r *Registers) GetIME() bool {
	return r.ime
}

func (r *Registers) SetIME(value bool) {
	r.ime = value
}

func (r *Registers) GetBC() uint16 {
	return r.bc
}

func (r *Registers) DecBC() {
	r.bc--
}

func (r *Registers) IncBC() {
	r.bc++
}

func (r *Registers) SetBC(input uint16) {
	r.bc = input
}

func (r *Registers) GetDE() uint16 {
	return r.de
}

func (r *Registers) DecDE() {
	r.de--
}

func (r *Registers) IncDE() {
	r.de++
}

func (r *Registers) SetDE(input uint16) {
	r.de = input
}

func (r *Registers) GetHL() uint16 {
	return r.hl
}

func (r *Registers) DecHL() {
	r.hl--
}

func (r *Registers) IncHL() {
	r.hl++
}

func (r *Registers) SetHL(input uint16) {
	r.hl = input
}

func (r *Registers) GetAF() uint16 {
	return r.af
}

func (r *Registers) SetAF(input uint16) {
	r.af = input & 0xFFF0
}

func (r *Registers) GetSP() uint16 {
	return r.sp
}

func (r *Registers) DecSP() {
	r.sp--
}

func (r *Registers) IncSP() {
	r.sp++
}

func (r *Registers) SetSP(input uint16) {
	r.sp = input
}

func (r *Registers) GetPC() uint16 {
	return r.pc
}

func (r *Registers) DecPC() {
	r.pc--
}

func (r *Registers) IncPC() {
	r.pc++
}
func (r *Registers) SetPC(input uint16) {
	r.pc = input
}

// Returns true if the flag is flipped, false otherwise
func (r *Registers) GetFlag(flag FlagMask) bool {
	if r.af&uint16(flag) > 0 {
		return true
	} else {
		return false
	}
}

func (r *Registers) GetReg(reg Reg) uint8 {
	switch reg {
	case A:
		return uint8((r.af & 0xFF00) >> 8)
	case F:
		return uint8(r.af & 0x00F0)
	case B:
		return uint8((r.bc & 0xFF00) >> 8)
	case C:
		return uint8(r.bc & 0x00FF)
	case D:
		return uint8((r.de & 0xFF00) >> 8)
	case E:
		return uint8(r.de & 0x00FF)
	case H:
		return uint8((r.hl & 0xFF00) >> 8)
	case L:
		return uint8(r.hl & 0x00FF)
	default:
		fmt.Println("Invalid register")
		return 0
	}
}

func (r *Registers) SetReg(reg Reg, input uint8) {
	switch reg {
	case A:
		r.af = uint16(input)<<8 | r.af&0x00F0
	case F:
		r.af = r.af&0xFF00 | uint16(input)&0x00F0
	case B:
		r.bc = uint16(input)<<8 | r.bc&0x00FF
	case C:
		r.bc = r.bc&0xFF00 | uint16(input)&0x00FF
	case D:
		r.de = uint16(input)<<8 | r.de&0x00FF
	case E:
		r.de = r.de&0xFF00 | uint16(input)&0x00FF
	case H:
		r.hl = uint16(input)<<8 | r.hl&0x00FF
	case L:
		r.hl = r.hl&0xFF00 | uint16(input)&0x00FF
	}
}

func (r *Registers) IncReg(reg Reg) {
	r.SetReg(reg, r.GetReg(reg) + 1)
}

func (r *Registers) DecReg(reg Reg) {
	r.SetReg(reg, r.GetReg(reg) - 1)
}

// Sets the given flag to true or false
func (r *Registers) SetFlag(flag FlagMask, flipped bool) {
	if flipped {
		r.af = r.af | uint16(flag)
	} else if !flipped {
		// F
		// 11110000
		// 01000000 SUBTRACTION_FLAG
		// (^SUBTRACTION_FLAG & 11110000) = 10110000
		// 11110000
		//&10110000
		// 10110000
		f := r.af & 0x00FF
		f = f & (^uint16(flag) & 0x00F0)
		r.af = (r.af & 0xFF00) | f
	}
}
