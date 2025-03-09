package cpu

import (
	"fmt"

	"github.com/phishbacon/gameboygo/common"
)

type Operation func(c *CPU) uint8
type AddrMode func(c *CPU)

type Instruction struct {
	Mnemonic  string
	Size      uint8
	Ticks     []uint8
	AddrMode  AddrMode
	Operation Operation
}

func (c *CPU) cpuCycles(cycles uint8) {
	var n int = int(cycles) * 4
	for i := 0; i < n; i++ {
		c.Ticks++
	}
	return
}

// no operation
func NONE(c *CPU) {
}

// load value pointed to by 16 bit address in to register R
func R_A16(c *CPU) {
	// grab low and hi byte from adddress pc and pc +1
	lo := c.bus.Read(c.Registers.PC)
	c.cpuCycles(1)
	hi := c.bus.Read(c.Registers.PC + 1)
	c.cpuCycles(1)
	c.Registers.PC += 2
	c.Fetched = (uint16(hi) << 8) | uint16(lo)
}

// load immediate 16 bit value into R
var R_N16 AddrMode = R_A16

// load register value into memory pointed to by 16 bit address
var A16_R AddrMode = R_A16

// grab signed 8 bit data
func E8(c *CPU) {
	// fmt.Printf("0x%04x relAddr: 0x%04x(%d)", c.registers.PC, c.bus.Read(c.registers.PC), int8(c.bus.Read(c.registers.PC)))
	c.RelAddr = int8(c.bus.Read(c.Registers.PC))
	c.Registers.PC++
	c.cpuCycles(1)
}

// load immediate 8 value into R
func R_N8(c *CPU) {
	lo := c.bus.Read(c.Registers.PC)
	c.cpuCycles(1)
	c.Registers.PC++
	c.Fetched = uint16(lo)
}

// load register value into high ram  address A8 + FF00
func A8_A(c *CPU) {
	lo := uint16(c.bus.Read(c.Registers.PC)) + 0xFF00
	c.cpuCycles(1)
	c.Registers.PC++
	c.Fetched = lo
}

// load value pointed to by high ram address into A
var A_A8 AddrMode = A8_A

//
// func A_A8(c *CPU) {
// 	lo := uint16(c.bus.Read(c.registers.GetPC())) + 0xFF00
// 	c.cpuCycles(1)
// 	c.registers.IncPC()
// 	c.Fetched = lo
// }

func HalfCarrySub(a uint8, b uint8) bool {
	return (int(a)&0x0f)-(int(b)&0x0f) < 0
}

func FullCarrySub(a uint8, b uint8) bool {
	return int(a)-int(b) < 0
}

func HalfCarrySbc(a uint8, b uint8, c uint8) bool {
	return ((int(a) & 0x0f) - (int(b) & 0x0f) - (int(c) & 0x0f)) < 0
}

func FullCarrySbc(a uint8, b uint8, c uint8) bool {
	return int(a)-int(b)-int(c) < 0
}

func HalfCarryAdd(a uint8, b uint8) bool {
	// a = 00001000 b = 00001000
	// 00001000
	//+00001000
	//=00010000

	// a & 0x0f = 00001000
	// b & 0x0f = 00001000
	//          +
	//            00010000
	// 00010000 & 00010000 = 00010000 // the 5th bit was flipped
	// 00010000 == 00010000
	// so we have a half carry
	return ((a&0x000F)+(b&0x000F))&0x0010 == 0x0010
}

func FullCarryAdd(a uint8, b uint8) bool {
	return uint16(a)+uint16(b) > 0x00FF
}

func HalfCarryAdc(a uint8, b uint8, c uint8) bool {
	return ((a&0x000F)+(b&0x000F)+(c&0x000F))&0x0010 == 0x0010
}

func FullCarryAdc(a uint8, b uint8, c uint8) bool {
	return uint16(a)+uint16(b)+uint16(c) > 0x00FF
}

func (c *CPU) SetDecRegFlags(register uint8) {
	c.Registers.SetFlag(common.SUBTRACTION_FLAG, true)
	c.Registers.SetFlag(common.ZERO_FLAG, register-1 == 0)
	c.Registers.SetFlag(common.HALF_CARRY_FLAG, register&0x000F == 0x0000)
}

func (c *CPU) SetIncRegFlags(register uint8) {
	c.Registers.SetFlag(common.SUBTRACTION_FLAG, false)
	c.Registers.SetFlag(common.ZERO_FLAG, register+1 == 0)
	c.Registers.SetFlag(common.HALF_CARRY_FLAG, register&0x000F == 0x000F)
}

func (c *CPU) SetDecFlags(value uint8) {
	c.Registers.SetFlag(common.SUBTRACTION_FLAG, true)
	c.Registers.SetFlag(common.ZERO_FLAG, value-1 == 0)
	c.Registers.SetFlag(common.HALF_CARRY_FLAG, value&0x000F == 0x0000)
}

func (c *CPU) SetIncFlags(value uint8) {
	c.Registers.SetFlag(common.SUBTRACTION_FLAG, false)
	c.Registers.SetFlag(common.ZERO_FLAG, value+1 == 0)
	c.Registers.SetFlag(common.HALF_CARRY_FLAG, value&0x000F == 0x000F)
}

func (c *CPU) SetRotateFlags(registerVal uint8, leftOrRight string) {
	c.Registers.SetFlag(common.ZERO_FLAG, false)
	c.Registers.SetFlag(common.SUBTRACTION_FLAG, false)
	c.Registers.SetFlag(common.HALF_CARRY_FLAG, false)

	switch leftOrRight {
	case "L":
		carryBit := registerVal >> 7
		if carryBit == 0 {
			c.Registers.SetFlag(common.CARRY_FLAG, false)
		} else if carryBit == 1 {
			c.Registers.SetFlag(common.CARRY_FLAG, true)
		}
	case "R":
		carryBit := registerVal & 0x0001
		if carryBit == 0 {
			c.Registers.SetFlag(common.CARRY_FLAG, false)
		} else if carryBit == 1 {
			c.Registers.SetFlag(common.CARRY_FLAG, true)
		}
	}
}
func (c *CPU) SetCBRotateFlags(registerVal uint8, leftOrRight string, throughCarry bool) uint8 {
	c.Registers.SetFlag(common.SUBTRACTION_FLAG, false)
	c.Registers.SetFlag(common.HALF_CARRY_FLAG, false)
	var oldCarry uint8
	if c.Registers.GetFlag(common.CARRY_FLAG) {
		oldCarry = 1
	}

	switch leftOrRight {
	case "L":
		carryBit := registerVal >> 7
		if carryBit == 0 {
			c.Registers.SetFlag(common.CARRY_FLAG, false)
		} else if carryBit == 1 {
			c.Registers.SetFlag(common.CARRY_FLAG, true)
		}
		if throughCarry {
			registerVal = (registerVal << 1) | oldCarry
		} else {
			registerVal = (registerVal << 1) | (registerVal >> 7)
		}
	case "R":
		carryBit := registerVal & 0x0001
		if carryBit == 0 {
			c.Registers.SetFlag(common.CARRY_FLAG, false)
		} else if carryBit == 1 {
			c.Registers.SetFlag(common.CARRY_FLAG, true)
		}
		if throughCarry {
			registerVal = (registerVal >> 1) | (oldCarry << 7)
		} else {
			registerVal = (registerVal >> 1) | (registerVal << 7)
		}
	}
	c.Registers.SetFlag(common.ZERO_FLAG, registerVal == 0)
	return registerVal
}

func (c *CPU) SetShiftFlags(registerVal uint8, leftOrRight string, logically bool) uint8 {
	c.Registers.SetFlag(common.SUBTRACTION_FLAG, false)
	c.Registers.SetFlag(common.HALF_CARRY_FLAG, false)

	switch leftOrRight {
	case "L":
		carryBit := registerVal >> 7
		if carryBit == 0 {
			c.Registers.SetFlag(common.CARRY_FLAG, false)
		} else if carryBit == 1 {
			c.Registers.SetFlag(common.CARRY_FLAG, true)
		}
		registerVal <<= 1
	case "R":
		carryBit := registerVal & 0x0001
		if carryBit == 0 {
			c.Registers.SetFlag(common.CARRY_FLAG, false)
		} else if carryBit == 1 {
			c.Registers.SetFlag(common.CARRY_FLAG, true)
		}
		if logically {
			registerVal >>= 1
		} else {
			// 11000110 >> 1 =       01100011
			// 11000110 & 10000000 = 10000000
			registerVal = (registerVal >> 1) | (registerVal & 0x80)
		}
	}
	c.Registers.SetFlag(common.ZERO_FLAG, registerVal == 0)
	return registerVal
}

func (c *CPU) SetSwapFlags(registerVal uint8) uint8 {
	c.Registers.SetFlag(common.SUBTRACTION_FLAG, false)
	c.Registers.SetFlag(common.HALF_CARRY_FLAG, false)
	c.Registers.SetFlag(common.CARRY_FLAG, false)

	highNibble := registerVal >> 4 & 0x000F
	lowNibble := registerVal & 0x000F

	return (lowNibble << 4) | highNibble
}

func (c *CPU) SetBitFlags(registerVal uint8, bit uint8) {
	c.Registers.SetFlag(common.SUBTRACTION_FLAG, false)
	c.Registers.SetFlag(common.HALF_CARRY_FLAG, true)
	var i uint8
	var bitValue uint8 = 1
	for i = 0; i < bit; i++ {
		bitValue *= 2
	}
	// 00101110
	//&00010000
	// 00000000
	c.Registers.SetFlag(common.ZERO_FLAG, registerVal&bitValue == 0)
}
func (c *CPU) SetBit(registerVal uint8, bit uint8, set bool) uint8 {
	var i uint8
	var bitValue uint8 = 1
	for i = 0; i < bit; i++ {
		bitValue *= 2
	}
	if set {
		if registerVal&bitValue == 0 {
			registerVal &= bitValue
		}
	} else {
		//
		// 11000100
		//&01000000
		//=01000000 != 0 so
		// 11000100
		//^01000000
		//=10000100
		if registerVal&bitValue != 0 {
			registerVal ^= bitValue
		}
	}
	return registerVal
}

func HalfCarryAdd16(a uint16, b uint16) bool {
	// a 0000111000000000
	// b 0000001000000000
	// a & 0x0FFF = 0000111000000000
	// b & 0x0FFF = 0000001000000000
	//            +=0001000000000000
	// 0001000000000000 & 0x1000 = 0001000000000000
	// overflow from bit 11
	return ((a&0x0FFF)+(b&0x0FFF))&0x0100 == 0x0100
}

func FullCarryAdd16(a uint16, b uint16) bool {
	return uint32(a)+uint32(b) > 0xFFFF
}

func (c *CPU) SetAddFlags(a, b uint8) {
	c.Registers.SetFlag(common.ZERO_FLAG, a+b == 0)
	c.Registers.SetFlag(common.SUBTRACTION_FLAG, false)
	c.Registers.SetFlag(common.HALF_CARRY_FLAG, HalfCarryAdd(a, b))
	c.Registers.SetFlag(common.CARRY_FLAG, FullCarryAdd(a, b))
}

func (c *CPU) SetSubFlags(a, b uint8) {
	c.Registers.SetFlag(common.ZERO_FLAG, a-b == 0)
	c.Registers.SetFlag(common.SUBTRACTION_FLAG, true)
	c.Registers.SetFlag(common.HALF_CARRY_FLAG, HalfCarrySub(a, b))
	c.Registers.SetFlag(common.CARRY_FLAG, FullCarrySub(a, b))
}
func (c *CPU) SetCpFlags(a, b uint8) {
	c.Registers.SetFlag(common.ZERO_FLAG, a-b == 0)
	c.Registers.SetFlag(common.SUBTRACTION_FLAG, true)
	c.Registers.SetFlag(common.HALF_CARRY_FLAG, HalfCarrySub(a, b))
	c.Registers.SetFlag(common.CARRY_FLAG, b > a)
}

func (c *CPU) SetAdcFlags(a, b uint8) uint8 {
	var carryFlag uint8
	if c.Registers.GetFlag(common.CARRY_FLAG) {
		carryFlag = 1
	}
	c.Registers.SetFlag(common.ZERO_FLAG, (uint16(a)+uint16(b)+uint16(carryFlag))&0x00FF == 0)
	c.Registers.SetFlag(common.SUBTRACTION_FLAG, false)
	c.Registers.SetFlag(common.HALF_CARRY_FLAG, HalfCarryAdc(a, b, carryFlag))
	c.Registers.SetFlag(common.CARRY_FLAG, FullCarryAdc(a, b, carryFlag))

	return carryFlag
}

func (c *CPU) SetSbcFlags(a, b uint8) uint8 {
	var carryFlag uint8
	if c.Registers.GetFlag(common.CARRY_FLAG) {
		carryFlag = 1
	}
	c.Registers.SetFlag(common.ZERO_FLAG, uint16(a)-uint16(b)-uint16(carryFlag) == 0)
	c.Registers.SetFlag(common.SUBTRACTION_FLAG, true)
	c.Registers.SetFlag(common.HALF_CARRY_FLAG, HalfCarrySbc(a, b, carryFlag))
	c.Registers.SetFlag(common.CARRY_FLAG, FullCarrySbc(a, b, carryFlag))

	return carryFlag
}
func (c *CPU) SetAddFlags16(a uint16, b uint16) {
	c.Registers.SetFlag(common.SUBTRACTION_FLAG, false)
	c.Registers.SetFlag(common.HALF_CARRY_FLAG, HalfCarryAdd16(a, b))
	c.Registers.SetFlag(common.CARRY_FLAG, FullCarryAdd16(a, b))
}
func (c *CPU) SetAndFlags(a uint8) {
	c.Registers.SetFlag(common.ZERO_FLAG, a == 0)
	c.Registers.SetFlag(common.SUBTRACTION_FLAG, false)
	c.Registers.SetFlag(common.HALF_CARRY_FLAG, true)
	c.Registers.SetFlag(common.CARRY_FLAG, false)
}
func (c *CPU) SetXorFlags(a uint8) {
	c.Registers.SetFlag(common.ZERO_FLAG, a == 0)
	c.Registers.SetFlag(common.SUBTRACTION_FLAG, false)
	c.Registers.SetFlag(common.HALF_CARRY_FLAG, false)
	c.Registers.SetFlag(common.CARRY_FLAG, false)
}

var Instructions = [0x0100]Instruction{
	0x00: {
		Mnemonic: "NOP",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) uint8 {
			return 0
		},
	},
	0x01: {
		Mnemonic: "LD_BC_N16",
		Size:     3,
		Ticks:    []uint8{12},
		AddrMode: R_A16,
		Operation: func(c *CPU) uint8 {
			c.Registers.SetBC(c.Fetched)
			return 0
		},
	},
	0x02: {
		Mnemonic: "LD_[BC]_A",
		Size:     1,
		Ticks:    []uint8{8},
		AddrMode: NONE,
		Operation: func(c *CPU) uint8 {
			c.bus.Write(c.Registers.GetBC(), c.Registers.A)
			c.cpuCycles(1)
			return 0
		},
	},
	0x03: {
		Mnemonic: "INC_BC",
		Size:     1,
		Ticks:    []uint8{8},
		AddrMode: NONE,
		Operation: func(c *CPU) uint8 {
			c.Registers.SetBC(c.Registers.GetBC() + 1)
			c.cpuCycles(1)
			return 0
		},
	},
	0x04: {
		Mnemonic: "INC_B",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) uint8 {
			c.SetIncRegFlags(c.Registers.B)
			c.Registers.B++
			return 0
		},
	},
	0x05: {
		Mnemonic: "DEC_B",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) uint8 {
			c.SetDecRegFlags(c.Registers.B)
			c.Registers.B--
			return 0
		},
	},
	0x06: {
		Mnemonic: "LD_B_N8",
		Size:     2,
		Ticks:    []uint8{8},
		AddrMode: R_N8,
		Operation: func(c *CPU) uint8 {
			c.Registers.B = uint8(c.Fetched)
			return 0
		},
	},
	0x07: {
		Mnemonic: "RLCA",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) uint8 {
			a := c.Registers.A
			c.SetRotateFlags(a, "L")
			// a = 11101000
			// a << 1 = 11010000
			// a >> 7 = 00000001
			//          11010001
			c.Registers.A = (a << 1) | (a >> 7)
			return 0
		},
	},
	0x08: {
		Mnemonic: "LD_[A16]_SP",
		Size:     3,
		Ticks:    []uint8{20},
		AddrMode: R_A16,
		Operation: func(c *CPU) uint8 {
			lo := uint8(c.Registers.SP & 0x00FF)
			hi := uint8(c.Registers.SP >> 8)
			c.bus.Write(c.Fetched, lo)
			c.cpuCycles(1)
			c.bus.Write(c.Fetched+1, hi)
			c.cpuCycles(1)
			return 0
		},
	},
	0x09: {
		Mnemonic: "ADD_HL_BC",
		Size:     1,
		Ticks:    []uint8{8},
		AddrMode: NONE,
		Operation: func(c *CPU) uint8 {
			c.SetAddFlags16(c.Registers.GetHL(), c.Registers.GetBC())
			c.Registers.SetHL(c.Registers.GetHL() + c.Registers.GetBC())
			c.cpuCycles(1)
			return 0
		},
	},
	0x0A: {
		Mnemonic: "LD_A_[BC]",
		Size:     1,
		Ticks:    []uint8{8},
		AddrMode: NONE,
		Operation: func(c *CPU) uint8 {
			c.Registers.A = c.bus.Read(c.Registers.GetBC())
			c.cpuCycles(1)
			return 0
		},
	},
	0x0B: {
		Mnemonic: "DEC_BC",
		Size:     1,
		Ticks:    []uint8{8},
		AddrMode: NONE,
		Operation: func(c *CPU) uint8 {
			c.Registers.SetBC(c.Registers.GetBC() - 1)
			c.cpuCycles(1)
			return 0
		},
	},
	0x0C: {
		Mnemonic: "INC_C",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) uint8 {
			c.SetIncRegFlags(c.Registers.C)
			c.Registers.C++
			return 0
		},
	},
	0x0D: {
		Mnemonic: "DEC_C",
		Size:     1,
		AddrMode: NONE,
		Ticks:    []uint8{4},
		Operation: func(c *CPU) uint8 {
			c.SetDecRegFlags(c.Registers.C)
			c.Registers.C--
			return 0
		},
	},
	0x0E: {
		Mnemonic: "LD_C_N8",
		Size:     2,
		Ticks:    []uint8{8},
		AddrMode: R_N8,
		Operation: func(c *CPU) uint8 {
			c.Registers.C = uint8(c.Fetched)
			return 0
		},
	},
	0x0F: {
		Mnemonic: "RRCA",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) uint8 {
			a := c.Registers.A
			c.SetRotateFlags(a, "R")
			// a = 11101001
			// a >> 1 = 01110100
			// a << 7 = 10000000
			//          11110100
			c.Registers.A = (a >> 1) | (a << 7)
			return 0
		},
	},
	0x10: {
		Mnemonic: "STOP_N8",
		Size:     2,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) uint8 {
			fmt.Println("STOP!\n")
			return 0
		},
	},
	0x11: {
		Mnemonic: "LD_DE_N16",
		Size:     3,
		Ticks:    []uint8{12},
		AddrMode: R_N16,
		Operation: func(c *CPU) uint8 {
			c.Registers.SetDE(c.Fetched)
			return 0
		},
	},
	0x12: {
		Mnemonic: "LD_[DE]_A",
		Size:     1,
		Ticks:    []uint8{8},
		AddrMode: NONE,
		Operation: func(c *CPU) uint8 {
			c.bus.Write(c.Registers.GetDE(), c.Registers.A)
			c.cpuCycles(1)
			return 0
		},
	},
	0x13: {
		Mnemonic: "INC_DE",
		Size:     1,
		Ticks:    []uint8{8},
		AddrMode: NONE,
		Operation: func(c *CPU) uint8 {
			c.Registers.SetDE(c.Registers.GetDE() + 1)
			c.cpuCycles(1)
			return 0
		},
	},
	0x14: {
		Mnemonic: "INC_D",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) uint8 {
			c.SetIncRegFlags(c.Registers.D)
			c.Registers.D++
			return 0
		},
	},
	0x15: {
		Mnemonic: "DEC_D",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) uint8 {
			c.SetDecRegFlags(c.Registers.D)
			c.Registers.D--
			return 0
		},
	},
	0x16: {
		Mnemonic: "LD_D_N8",
		Size:     2,
		Ticks:    []uint8{8},
		AddrMode: R_N8,
		Operation: func(c *CPU) uint8 {
			c.Registers.D = uint8(c.Fetched)
			return 0
		},
	},
	0x17: {
		Mnemonic: "RLA",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) uint8 {
			a := c.Registers.A
			var oldCarry uint8
			if c.Registers.GetFlag(common.CARRY_FLAG) {
				oldCarry = 1
			}
			c.SetRotateFlags(a, "L")
			// oldCarry = 1
			// a = 10010100
			// a << 1 = 00101000
			// a | 00000001 = 00101001
			c.Registers.A = (a << 1) | oldCarry
			return 0
		},
	},
	0x18: {
		Mnemonic: "JR_E8",
		Size:     2,
		Ticks:    []uint8{12},
		AddrMode: E8,
		Operation: func(c *CPU) uint8 {
			c.Registers.PC = uint16(int16(c.Registers.PC) + int16(c.RelAddr))
			c.cpuCycles(1)
			return 0
		},
	},
	0x19: {
		Mnemonic: "ADD_HL_DE",
		Size:     1,
		Ticks:    []uint8{8},
		AddrMode: NONE,
		Operation: func(c *CPU) uint8 {
			c.SetAddFlags16(c.Registers.GetHL(), c.Registers.GetDE())
			c.Registers.SetHL(c.Registers.GetHL() + c.Registers.GetDE())
			c.cpuCycles(1)
			return 0
		},
	},
	0x1A: {
		Mnemonic: "LD_A_[DE]",
		Size:     1,
		Ticks:    []uint8{8},
		AddrMode: NONE,
		Operation: func(c *CPU) uint8 {
			fetched := c.bus.Read(c.Registers.GetDE())
			c.cpuCycles(1)
			c.Registers.A = fetched
			return 0
		},
	},
	0x1B: {
		Mnemonic: "DEC_DE",
		Size:     1,
		Ticks:    []uint8{8},
		AddrMode: NONE,
		Operation: func(c *CPU) uint8 {
			c.Registers.SetDE(c.Registers.GetDE() - 1)
			c.cpuCycles(1)
			return 0
		},
	},
	0x1C: {
		Mnemonic: "INC_E",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) uint8 {
			c.SetIncRegFlags(c.Registers.E)
			c.Registers.E++
			return 0
		},
	},
	0x1D: {
		Mnemonic: "DEC_E",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) uint8 {
			c.SetDecRegFlags(c.Registers.E)
			c.Registers.E--
			return 0
		},
	},
	0x1E: {
		Mnemonic: "LD_E_N8",
		Size:     2,
		Ticks:    []uint8{8},
		AddrMode: R_N8,
		Operation: func(c *CPU) uint8 {
			c.Registers.E = uint8(c.Fetched)
			return 0
		},
	},
	0x1F: {
		Mnemonic: "RRA",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) uint8 {
			a := c.Registers.A
			var oldCarry uint8
			if c.Registers.GetFlag(common.CARRY_FLAG) {
				oldCarry = 1
			}
			c.SetRotateFlags(a, "R")
			// oldCarry = 1
			// a = 10010100
			// a >> 1 = 01001010
			// oldCarry << 7 = 10000000
			// a | 10000000 = 11001010
			c.Registers.A = (a >> 1) | (oldCarry << 7)
			return 0
		},
	},
	0x20: {
		Mnemonic: "JR_NZ_E8",
		Size:     2,
		Ticks:    []uint8{12, 8},
		AddrMode: E8,
		Operation: func(c *CPU) uint8 {
			if !c.Registers.GetFlag(common.ZERO_FLAG) {
				c.Registers.PC = uint16(int16(c.Registers.PC) + int16(c.RelAddr))
				c.cpuCycles(1)
				return 0
			}
			return 1
		},
	},
	0x21: {
		Mnemonic: "LD_HL_N16",
		Size:     3,
		Ticks:    []uint8{12},
		AddrMode: R_A16,
		Operation: func(c *CPU) uint8 {
			c.Registers.SetHL(c.Fetched)
			return 0
		},
	},
	0x22: {
		Mnemonic: "LD_[HLI]_A",
		Size:     1,
		Ticks:    []uint8{8},
		AddrMode: NONE,
		Operation: func(c *CPU) uint8 {
			hl := c.Registers.GetHL()
			c.bus.Write(hl, c.Registers.A)
			c.Registers.SetHL(hl + 1)
			c.cpuCycles(1)
			return 0
		},
	},
	0x23: {
		Mnemonic: "INC_HL",
		Size:     1,
		Ticks:    []uint8{8},
		AddrMode: NONE,
		Operation: func(c *CPU) uint8 {
			c.Registers.SetHL(c.Registers.GetHL() + 1)
			c.cpuCycles(1)
			return 0
		},
	},
	0x24: {
		Mnemonic: "INC_H",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) uint8 {
			c.SetIncRegFlags(c.Registers.H)
			c.Registers.H++
			return 0
		},
	},
	0x25: {
		Mnemonic: "DEC_H",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) uint8 {
			c.SetDecRegFlags(c.Registers.H)
			c.Registers.H--
			return 0
		},
	},
	0x26: {
		Mnemonic: "LD_H_N8",
		Size:     2,
		Ticks:    []uint8{8},
		AddrMode: R_N8,
		Operation: func(c *CPU) uint8 {
			c.Registers.H = uint8(c.Fetched)
			return 0
		},
	},
	0x27: {
		Mnemonic: "DAA",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) uint8 {
			var adj uint8 = 0
			carryFlag := false
			if c.Registers.GetFlag(common.SUBTRACTION_FLAG) {
				if c.Registers.GetFlag(common.HALF_CARRY_FLAG) {
					adj += 0x0006
				}
				if c.Registers.GetFlag(common.CARRY_FLAG) {
					adj += 0x0060
				}
				c.Registers.A -= adj
			} else {
				if c.Registers.GetFlag(common.HALF_CARRY_FLAG) || c.Registers.A&0x000F > 0x0009 {
					adj += 0x0006
				}
				if c.Registers.GetFlag(common.CARRY_FLAG) || c.Registers.A > 0x0099 {
					carryFlag = true
					adj += 0x0060
				}
				c.Registers.A += adj
			}
			c.Registers.SetFlag(common.HALF_CARRY_FLAG, false)
			c.Registers.SetFlag(common.ZERO_FLAG, c.Registers.A == 0)
			c.Registers.SetFlag(common.CARRY_FLAG, carryFlag)
			return 0
		},
	},
	0x28: {
		Mnemonic: "JR_Z_E8",
		Size:     2,
		Ticks:    []uint8{12, 8},
		AddrMode: E8,
		Operation: func(c *CPU) uint8 {
			if c.Registers.GetFlag(common.ZERO_FLAG) {
				c.Registers.PC = uint16(int16(c.Registers.PC) + int16(c.RelAddr))
				c.cpuCycles(1)
				return 0
			}
			return 1
		},
	},
	0x29: {
		Mnemonic: "ADD_HL_HL",
		Size:     1,
		Ticks:    []uint8{8},
		AddrMode: NONE,
		Operation: func(c *CPU) uint8 {
			hl := c.Registers.GetHL()
			c.SetAddFlags16(hl, hl)
			c.Registers.SetHL(hl + hl)
			c.cpuCycles(1)
			return 0
		},
	},
	0x2A: {
		Mnemonic: "LD_A_[HLI]",
		Size:     1,
		Ticks:    []uint8{8},
		AddrMode: NONE,
		Operation: func(c *CPU) uint8 {
			fetched := c.bus.Read(c.Registers.GetHL())
			c.cpuCycles(1)
			c.Registers.A = fetched
			c.Registers.SetHL(c.Registers.GetHL() + 1)
			return 0
		},
	},
	0x2B: {
		Mnemonic: "DEC_HL",
		Size:     1,
		Ticks:    []uint8{8},
		AddrMode: NONE,
		Operation: func(c *CPU) uint8 {
			c.Registers.SetDE(c.Registers.GetDE() - 1)
			c.cpuCycles(1)
			return 0
		},
	},
	0x2C: {
		Mnemonic: "INC_L",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) uint8 {
			c.SetIncRegFlags(c.Registers.L)
			c.Registers.L++
			return 0
		},
	},
	0x2D: {
		Mnemonic: "DEC_L",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) uint8 {
			c.SetDecRegFlags(c.Registers.L)
			c.Registers.L--
			return 0
		},
	},
	0x2E: {
		Mnemonic: "LD_L_N8",
		Size:     2,
		Ticks:    []uint8{8},
		AddrMode: R_N8,
		Operation: func(c *CPU) uint8 {
			c.Registers.L = uint8(c.Fetched)
			return 0
		},
	},
	0x2F: {
		Mnemonic: "CPL",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) uint8 {
			c.Registers.A = ^c.Registers.A
			c.Registers.SetFlag(common.SUBTRACTION_FLAG, true)
			c.Registers.SetFlag(common.HALF_CARRY_FLAG, true)
			return 0
		},
	},
	0x30: {
		Mnemonic: "JR_NC_E8",
		Size:     2,
		Ticks:    []uint8{12, 8},
		AddrMode: E8,
		Operation: func(c *CPU) uint8 {
			if !c.Registers.GetFlag(common.CARRY_FLAG) {
				c.Registers.PC = uint16(int16(c.Registers.PC) + int16(c.RelAddr))
				c.cpuCycles(1)
				return 0
			}
			return 1
		},
	},
	0x31: {
		Mnemonic: "LD_SP_N16",
		Size:     3,
		Ticks:    []uint8{12},
		AddrMode: R_N16,
		Operation: func(c *CPU) uint8 {
			c.Registers.SP = c.Fetched
			return 0
		},
	},
	0x32: {
		Mnemonic: "LD_[HLD]_A",
		Size:     1,
		Ticks:    []uint8{8},
		AddrMode: NONE,
		Operation: func(c *CPU) uint8 {
			hl := c.Registers.GetHL()
			c.bus.Write(hl, c.Registers.A)
			c.Registers.SetHL(hl - 1)
			c.cpuCycles(1)
			return 0
		},
	},
	0x33: {
		Mnemonic: "INC_SP",
		Size:     1,
		Ticks:    []uint8{8},
		AddrMode: NONE,
		Operation: func(c *CPU) uint8 {
			c.Registers.SP++
			c.cpuCycles(1)
			return 0
		},
	},
	0x34: {
		Mnemonic: "INC_[HL]",
		Size:     1,
		Ticks:    []uint8{12},
		AddrMode: NONE,
		Operation: func(c *CPU) uint8 {
			val := c.bus.Read(c.Registers.GetHL())
			c.cpuCycles(1)
			c.SetIncFlags(val)
			c.bus.Write(c.Registers.GetHL(), val+1)
			c.cpuCycles(1)
			return 0
		},
	},
	0x35: {
		Mnemonic: "DEC_[HL]",
		Size:     1,
		Ticks:    []uint8{12},
		AddrMode: NONE,
		Operation: func(c *CPU) uint8 {
			val := c.bus.Read(c.Registers.GetHL())
			c.cpuCycles(1)
			c.SetDecFlags(val)
			c.bus.Write(c.Registers.GetHL(), val-1)
			c.cpuCycles(1)
			return 0
		},
	},
	0x36: {
		Mnemonic: "LD_[HL]_N8",
		Size:     2,
		Ticks:    []uint8{12},
		AddrMode: R_N8,
		Operation: func(c *CPU) uint8 {
			c.bus.Write(c.Registers.GetHL(), uint8(c.Fetched))
			c.cpuCycles(1)
			return 0
		},
	},
	0x37: {
		Mnemonic: "SCF",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) uint8 {
			c.Registers.SetFlag(common.SUBTRACTION_FLAG, false)
			c.Registers.SetFlag(common.HALF_CARRY_FLAG, false)
			c.Registers.SetFlag(common.CARRY_FLAG, true)
			return 0
		},
	},
	0x38: {
		Mnemonic: "JR_C_E8",
		Size:     2,
		Ticks:    []uint8{12, 8},
		AddrMode: E8,
		Operation: func(c *CPU) uint8 {
			if c.Registers.GetFlag(common.CARRY_FLAG) {
				c.Registers.PC = uint16(int16(c.Registers.PC) + int16(c.RelAddr))
				c.cpuCycles(1)
				return 0
			}
			return 1
		},
	},
	0x39: {
		Mnemonic: "ADD_HL_SP",
		Size:     1,
		Ticks:    []uint8{8},
		AddrMode: NONE,
		Operation: func(c *CPU) uint8 {
			hl := c.Registers.GetHL()
			c.SetAddFlags16(hl, c.Registers.SP)
			c.Registers.SetHL(hl + c.Registers.SP)
			c.cpuCycles(1)
			return 0
		},
	},
	0x3A: {
		Mnemonic: "LD_A_[HLD]",
		Size:     1,
		Ticks:    []uint8{8},
		AddrMode: NONE,
		Operation: func(c *CPU) uint8 {
			fetched := c.bus.Read(c.Registers.GetHL())
			c.cpuCycles(1)
			c.Registers.A = fetched
			c.Registers.SetHL(c.Registers.GetHL() - 1)
			return 0
		},
	},
	0x3B: {
		Mnemonic: "DEC_SP",
		Size:     1,
		Ticks:    []uint8{8},
		AddrMode: NONE,
		Operation: func(c *CPU) uint8 {
			c.Registers.SP--
			c.cpuCycles(1)
			return 0
		},
	},
	0x3C: {
		Mnemonic: "INC_A",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) uint8 {
			c.SetIncRegFlags(c.Registers.A)
			c.Registers.A++
			return 0
		},
	},
	0x3D: {
		Mnemonic: "DEC_A",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) uint8 {
			c.SetDecRegFlags(c.Registers.A)
			c.Registers.A--
			return 0
		},
	},
	0x3E: {
		Mnemonic: "LD_A_N8",
		Size:     2,
		Ticks:    []uint8{8},
		AddrMode: R_N8,
		Operation: func(c *CPU) uint8 {
			c.Registers.A = uint8(c.Fetched)
			return 0
		},
	},
	0x3F: {
		Mnemonic: "CCF",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) uint8 {
			c.Registers.SetFlag(common.SUBTRACTION_FLAG, false)
			c.Registers.SetFlag(common.HALF_CARRY_FLAG, false)
			og := c.Registers.GetFlag(common.CARRY_FLAG)
			c.Registers.SetFlag(common.CARRY_FLAG, !og)
			return 0
		},
	},
	0x40: {
		Mnemonic: "LD_B_B",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) uint8 {
			return 0
		},
	},
	0x41: {
		Mnemonic: "LD_B_C",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) uint8 {
			c.Registers.B = c.Registers.C
			return 0
		},
	},
	0x42: {
		Mnemonic: "LD_B_D",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) uint8 {
			c.Registers.B = c.Registers.D
			return 0
		},
	},
	0x43: {
		Mnemonic: "LD_B_E",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) uint8 {
			c.Registers.B = c.Registers.E
			return 0
		},
	},
	0x44: {
		Mnemonic: "LD_B_H",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) uint8 {
			c.Registers.B = c.Registers.H
			return 0
		},
	},
	0x45: {
		Mnemonic: "LD_B_L",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) uint8 {
			c.Registers.B = c.Registers.L
			return 0
		},
	},
	0x46: {
		Mnemonic: "LD_B_[HL]",
		Size:     1,
		Ticks:    []uint8{8},
		AddrMode: NONE,
		Operation: func(c *CPU) uint8 {
			val := c.bus.Read(c.Registers.GetHL())
			c.cpuCycles(1)
			c.Registers.B = val
			return 0
		},
	},
	0x47: {
		Mnemonic: "LD_B_A",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) uint8 {
			c.Registers.B = c.Registers.A
			return 0
		},
	},
	0x48: {
		Mnemonic: "LD_C_B",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) uint8 {
			c.Registers.C = c.Registers.B
			return 0
		},
	},
	0x49: {
		Mnemonic: "LD_C_C",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) uint8 {
			return 0
		},
	},
	0x4A: {
		Mnemonic: "LD_C_D",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) uint8 {
			c.Registers.C = c.Registers.D
			return 0
		},
	},
	0x4B: {
		Mnemonic: "LD_C_E",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) uint8 {
			c.Registers.C = c.Registers.E
			return 0
		},
	},
	0x4C: {
		Mnemonic: "LD_C_H",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) uint8 {
			c.Registers.C = c.Registers.H
			return 0
		},
	},
	0x4D: {
		Mnemonic: "LD_C_L",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) uint8 {
			c.Registers.C = c.Registers.L
			return 0
		},
	},
	0x4E: {
		Mnemonic: "LD_C_[HL]",
		Size:     1,
		Ticks:    []uint8{8},
		AddrMode: NONE,
		Operation: func(c *CPU) uint8 {
			val := c.bus.Read(c.Registers.GetHL())
			c.cpuCycles(1)
			c.Registers.C = val
			return 0
		},
	},
	0x4F: {
		Mnemonic: "LD_C_A",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) uint8 {
			c.Registers.C = c.Registers.A
			return 0
		},
	},
	0x50: {
		Mnemonic: "LD_D_B",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) uint8 {
			c.Registers.D = c.Registers.B
			return 0
		},
	},
	0x51: {
		Mnemonic: "LD_D_C",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) uint8 {
			c.Registers.D = c.Registers.C
			return 0
		},
	},
	0x52: {
		Mnemonic: "LD_D_D",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) uint8 {
			return 0
		},
	},
	0x53: {
		Mnemonic: "LD_D_E",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) uint8 {
			c.Registers.D = c.Registers.E
			return 0
		},
	},
	0x54: {
		Mnemonic: "LD_D_H",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) uint8 {
			c.Registers.D = c.Registers.H
			return 0
		},
	},
	0x55: {
		Mnemonic: "LD_D_L",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) uint8 {
			c.Registers.D = c.Registers.L
			return 0
		},
	},
	0x56: {
		Mnemonic: "LD_D_[HL]",
		Size:     1,
		Ticks:    []uint8{8},
		AddrMode: NONE,
		Operation: func(c *CPU) uint8 {
			val := c.bus.Read(c.Registers.GetHL())
			c.cpuCycles(1)
			c.Registers.D = val
			return 0
		},
	},
	0x57: {
		Mnemonic: "LD_D_A",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) uint8 {
			c.Registers.D = c.Registers.A
			return 0
		},
	},
	0x58: {
		Mnemonic: "LD_E_B",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) uint8 {
			c.Registers.E = c.Registers.B
			return 0
		},
	},
	0x59: {
		Mnemonic: "LD_E_C",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) uint8 {
			c.Registers.E = c.Registers.C
			return 0
		},
	},
	0x5A: {
		Mnemonic: "LD_E_D",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) uint8 {
			c.Registers.E = c.Registers.D
			return 0
		},
	},
	0x5B: {
		Mnemonic: "LD_E_E",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) uint8 {
			return 0
		},
	},
	0x5C: {
		Mnemonic: "LD_E_H",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) uint8 {
			c.Registers.E = c.Registers.H
			return 0
		},
	},
	0x5D: {
		Mnemonic: "LD_E_L",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) uint8 {
			c.Registers.E = c.Registers.L
			return 0
		},
	},
	0x5E: {
		Mnemonic: "LD_E_[HL]",
		Size:     1,
		Ticks:    []uint8{8},
		AddrMode: NONE,
		Operation: func(c *CPU) uint8 {
			val := c.bus.Read(c.Registers.GetHL())
			c.cpuCycles(1)
			c.Registers.E = val
			return 0
		},
	},
	0x5F: {
		Mnemonic: "LD_E_A",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) uint8 {
			c.Registers.E = c.Registers.A
			return 0
		},
	},
	0x60: {
		Mnemonic: "LD_H_B",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) uint8 {
			c.Registers.H = c.Registers.B
			return 0
		},
	},
	0x61: {
		Mnemonic: "LD_H_C",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) uint8 {
			c.Registers.H = c.Registers.C
			return 0
		},
	},
	0x62: {
		Mnemonic: "LD_H_D",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) uint8 {
			c.Registers.H = c.Registers.D
			return 0
		},
	},
	0x63: {
		Mnemonic: "LD_H_E",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) uint8 {
			c.Registers.H = c.Registers.E
			return 0
		},
	},
	0x64: {
		Mnemonic: "LD_H_H",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) uint8 {
			return 0
		},
	},
	0x65: {
		Mnemonic: "LD_H_L",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) uint8 {
			c.Registers.H = c.Registers.L
			return 0
		},
	},
	0x66: {
		Mnemonic: "LD_H_[HL]",
		Size:     1,
		Ticks:    []uint8{8},
		AddrMode: NONE,
		Operation: func(c *CPU) uint8 {
			val := c.bus.Read(c.Registers.GetHL())
			c.cpuCycles(1)
			c.Registers.H = val
			return 0
		},
	},
	0x67: {
		Mnemonic: "LD_H_A",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) uint8 {
			c.Registers.H = c.Registers.A
			return 0
		},
	},
	0x68: {
		Mnemonic: "LD_L_B",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) uint8 {
			c.Registers.L = c.Registers.B
			return 0
		},
	},
	0x69: {
		Mnemonic: "LD_L_C",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) uint8 {
			c.Registers.L = c.Registers.C
			return 0
		},
	},
	0x6A: {
		Mnemonic: "LD_L_D",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) uint8 {
			c.Registers.L = c.Registers.D
			return 0
		},
	},
	0x6B: {
		Mnemonic: "LD_L_E",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) uint8 {
			c.Registers.L = c.Registers.E
			return 0
		},
	},
	0x6C: {
		Mnemonic: "LD_L_H",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) uint8 {
			c.Registers.L = c.Registers.H
			return 0
		},
	},
	0x6D: {
		Mnemonic: "LD_L_L",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) uint8 {
			return 0
		},
	},
	0x6E: {
		Mnemonic: "LD_L_[HL]",
		Size:     1,
		Ticks:    []uint8{8},
		AddrMode: NONE,
		Operation: func(c *CPU) uint8 {
			val := c.bus.Read(c.Registers.GetHL())
			c.cpuCycles(1)
			c.Registers.L = val
			return 0
		},
	},
	0x6F: {
		Mnemonic: "LD_L_A",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) uint8 {
			c.Registers.L = c.Registers.A
			return 0
		},
	},
	0x70: {
		Mnemonic: "LD_[HL]_B",
		Size:     1,
		Ticks:    []uint8{8},
		AddrMode: NONE,
		Operation: func(c *CPU) uint8 {
			c.bus.Write(c.Registers.GetHL(), c.Registers.B)
			c.cpuCycles(1)
			return 0
		},
	},
	0x71: {
		Mnemonic: "LD_[HL]_C",
		Size:     1,
		Ticks:    []uint8{8},
		AddrMode: NONE,
		Operation: func(c *CPU) uint8 {
			c.bus.Write(c.Registers.GetHL(), c.Registers.C)
			c.cpuCycles(1)
			return 0
		},
	},
	0x72: {
		Mnemonic: "LD_[HL]_D",
		Size:     1,
		Ticks:    []uint8{8},
		AddrMode: NONE,
		Operation: func(c *CPU) uint8 {
			c.bus.Write(c.Registers.GetHL(), c.Registers.D)
			c.cpuCycles(1)
			return 0
		},
	},
	0x73: {
		Mnemonic: "LD_[HL]_E",
		Size:     1,
		Ticks:    []uint8{8},
		AddrMode: NONE,
		Operation: func(c *CPU) uint8 {
			c.bus.Write(c.Registers.GetHL(), c.Registers.E)
			c.cpuCycles(1)
			return 0
		},
	},
	0x74: {
		Mnemonic: "LD_[HL]_H",
		Size:     1,
		Ticks:    []uint8{8},
		AddrMode: NONE,
		Operation: func(c *CPU) uint8 {
			c.bus.Write(c.Registers.GetHL(), c.Registers.H)
			c.cpuCycles(1)
			return 0
		},
	},
	0x75: {
		Mnemonic: "LD_[HL]_L",
		Size:     1,
		Ticks:    []uint8{8},
		AddrMode: NONE,
		Operation: func(c *CPU) uint8 {
			c.bus.Write(c.Registers.GetHL(), c.Registers.L)
			c.cpuCycles(1)
			return 0
		},
	},
	0x76: {
		Mnemonic: "HALT",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) uint8 {
			c.Halted = true
			return 0
		},
	},
	0x77: {
		Mnemonic: "LD_[HL]_A",
		Size:     1,
		Ticks:    []uint8{8},
		AddrMode: NONE,
		Operation: func(c *CPU) uint8 {
			c.bus.Write(c.Registers.GetHL(), c.Registers.A)
			c.cpuCycles(1)
			return 0
		},
	},
	0x78: {
		Mnemonic: "LD_A_B",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) uint8 {
			c.Registers.A = c.Registers.B
			return 0
		},
	},
	0x79: {
		Mnemonic: "LD_A_C",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) uint8 {
			c.Registers.A = c.Registers.C
			return 0
		},
	},
	0x7A: {
		Mnemonic: "LD_A_D",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) uint8 {
			c.Registers.A = c.Registers.D
			return 0
		},
	},
	0x7B: {
		Mnemonic: "LD_A_E",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) uint8 {
			c.Registers.A = c.Registers.E
			return 0
		},
	},
	0x7C: {
		Mnemonic: "LD_A_H",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) uint8 {
			c.Registers.A = c.Registers.H
			return 0
		},
	},
	0x7D: {
		Mnemonic: "LD_A_L",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) uint8 {
			c.Registers.A = c.Registers.L
			return 0
		},
	},
	0x7E: {
		Mnemonic: "LD_A_[HL]",
		Size:     1,
		Ticks:    []uint8{8},
		AddrMode: NONE,
		Operation: func(c *CPU) uint8 {
			val := c.bus.Read(c.Registers.GetHL())
			c.cpuCycles(1)
			c.Registers.A = val
			return 0
		},
	},
	0x7F: {
		Mnemonic: "LD_A_A",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) uint8 {
			return 0
		},
	},
	0x80: {
		Mnemonic: "ADD_A_B",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) uint8 {
			a := c.Registers.A
			c.SetAddFlags(a, c.Registers.B)
			c.Registers.A += c.Registers.B
			return 0
		},
	},
	0x81: {
		Mnemonic: "ADD_A_C",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) uint8 {
			a := c.Registers.A
			c.SetAddFlags(a, c.Registers.C)
			c.Registers.A += c.Registers.C
			return 0
		},
	},
	0x82: {
		Mnemonic: "ADD_A_D",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) uint8 {
			a := c.Registers.A
			c.SetAddFlags(a, c.Registers.D)
			c.Registers.A += c.Registers.D
			return 0
		},
	},
	0x83: {
		Mnemonic: "ADD_A_E",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) uint8 {
			a := c.Registers.A
			c.SetAddFlags(a, c.Registers.E)
			c.Registers.A += c.Registers.E
			return 0
		},
	},
	0x84: {
		Mnemonic: "ADD_A_H",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) uint8 {
			a := c.Registers.A
			c.SetAddFlags(a, c.Registers.H)
			c.Registers.A += c.Registers.H
			return 0
		},
	},
	0x85: {
		Mnemonic: "ADD_A_L",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) uint8 {
			a := c.Registers.A
			c.SetAddFlags(a, c.Registers.L)
			c.Registers.A += c.Registers.L
			return 0
		},
	},
	0x86: {
		Mnemonic: "ADD_A_[HL]",
		Size:     1,
		Ticks:    []uint8{8},
		AddrMode: NONE,
		Operation: func(c *CPU) uint8 {
			val := c.bus.Read(c.Registers.GetHL())
			c.SetAddFlags(c.Registers.A, val)
			c.Registers.A += val
			return 0
		},
	},
	0x87: {
		Mnemonic: "ADD_A_A",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) uint8 {
			a := c.Registers.A
			c.SetAddFlags(a, a)
			c.Registers.A += a
			return 0
		},
	},
	0x88: {
		Mnemonic: "ADC_A_B",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) uint8 {
			a := c.Registers.A
			carryFlag := c.SetAdcFlags(a, c.Registers.B)
			c.Registers.A += (c.Registers.B) + carryFlag
			return 0
		},
	},
	0x89: {
		Mnemonic: "ADC_A_C",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) uint8 {
			a := c.Registers.A
			carryFlag := c.SetAdcFlags(a, c.Registers.C)
			c.Registers.A += (c.Registers.C) + carryFlag
			return 0
		},
	},
	0x8A: {
		Mnemonic: "ADC_A_D",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) uint8 {
			a := c.Registers.A
			carryFlag := c.SetAdcFlags(a, c.Registers.D)
			c.Registers.A += (c.Registers.D) + carryFlag
			return 0
		},
	},
	0x8B: {
		Mnemonic: "ADC_A_E",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) uint8 {
			a := c.Registers.A
			carryFlag := c.SetAdcFlags(a, c.Registers.E)
			c.Registers.A += (c.Registers.E) + carryFlag
			return 0
		},
	},
	0x8C: {
		Mnemonic: "ADC_A_H",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) uint8 {
			a := c.Registers.A
			carryFlag := c.SetAdcFlags(a, c.Registers.H)
			c.Registers.A += (c.Registers.H) + carryFlag
			return 0
		},
	},
	0x8D: {
		Mnemonic: "ADC_A_L",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) uint8 {
			a := c.Registers.A
			carryFlag := c.SetAdcFlags(a, c.Registers.L)
			c.Registers.A += (c.Registers.L) + carryFlag
			return 0
		},
	},
	0x8E: {
		Mnemonic: "ADC_A_[HL]",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) uint8 {
			val := c.bus.Read(c.Registers.GetHL())
			c.cpuCycles(1)
			carryFlag := c.SetAdcFlags(c.Registers.A, val)
			c.Registers.A += (val + carryFlag)
			return 0
		},
	},
	0x8F: {
		Mnemonic: "ADC_A_A",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) uint8 {
			a := c.Registers.A
			carryFlag := c.SetAdcFlags(a, a)
			c.Registers.A += (a + carryFlag)
			return 0
		},
	},
	0x90: {
		Mnemonic: "SUB_A_B",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) uint8 {
			a := c.Registers.A
			c.SetSubFlags(a, c.Registers.B)
			c.Registers.A -= c.Registers.B
			return 0
		},
	},
	0x91: {
		Mnemonic: "SUB_A_C",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) uint8 {
			a := c.Registers.A
			c.SetSubFlags(a, c.Registers.C)
			c.Registers.A -= c.Registers.C
			return 0
		},
	},
	0x92: {
		Mnemonic: "SUB_A_D",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) uint8 {
			a := c.Registers.A
			c.SetSubFlags(a, c.Registers.D)
			c.Registers.A -= c.Registers.D
			return 0
		},
	},
	0x93: {
		Mnemonic: "SUB_A_E",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) uint8 {
			a := c.Registers.A
			c.SetSubFlags(a, c.Registers.E)
			c.Registers.A -= c.Registers.E
			return 0
		},
	},
	0x94: {
		Mnemonic: "SUB_A_H",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) uint8 {
			a := c.Registers.A
			c.SetSubFlags(a, c.Registers.H)
			c.Registers.A -= c.Registers.H
			return 0
		},
	},
	0x95: {
		Mnemonic: "SUB_A_L",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) uint8 {
			a := c.Registers.A
			c.SetSubFlags(a, c.Registers.L)
			c.Registers.A -= c.Registers.L
			return 0
		},
	},
	0x96: {
		Mnemonic: "SUB_A_[HL]",
		Size:     1,
		Ticks:    []uint8{8},
		AddrMode: NONE,
		Operation: func(c *CPU) uint8 {
			val := c.bus.Read(c.Registers.GetHL())
			c.cpuCycles(1)
			a := c.Registers.A
			c.SetSubFlags(a, val)
			c.Registers.A -= a - val
			return 0
		},
	},
	0x97: {
		Mnemonic: "SUB_A_A",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) uint8 {
			c.Registers.SetFlag(common.ZERO_FLAG, true)
			c.Registers.SetFlag(common.SUBTRACTION_FLAG, true)
			c.Registers.SetFlag(common.HALF_CARRY_FLAG, false)
			c.Registers.SetFlag(common.CARRY_FLAG, false)
			c.Registers.A = 0
			return 0
		},
	},
	0x98: {
		Mnemonic: "SBC_A_B",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) uint8 {
			a := c.Registers.A
			carryFlag := c.SetSbcFlags(a, c.Registers.B)
			c.Registers.A -= c.Registers.B - carryFlag
			return 0
		},
	},
	0x99: {
		Mnemonic: "SBC_A_C",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) uint8 {
			a := c.Registers.A
			carryFlag := c.SetSbcFlags(a, c.Registers.C)
			c.Registers.A -= c.Registers.C - carryFlag
			return 0
		},
	},
	0x9A: {
		Mnemonic: "SBC_A_D",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) uint8 {
			a := c.Registers.A
			carryFlag := c.SetSbcFlags(a, c.Registers.D)
			c.Registers.A -= c.Registers.D - carryFlag
			return 0
		},
	},
	0x9B: {
		Mnemonic: "SBC_A_E",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) uint8 {
			a := c.Registers.A
			carryFlag := c.SetSbcFlags(a, c.Registers.E)
			c.Registers.A -= c.Registers.E - carryFlag
			return 0
		},
	},
	0x9C: {
		Mnemonic: "SBC_A_H",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) uint8 {
			a := c.Registers.A
			carryFlag := c.SetSbcFlags(a, c.Registers.H)
			c.Registers.A -= c.Registers.H - carryFlag
			return 0
		},
	},
	0x9D: {
		Mnemonic: "SBC_A_L",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) uint8 {
			a := c.Registers.A
			carryFlag := c.SetSbcFlags(a, c.Registers.L)
			c.Registers.A -= c.Registers.L - carryFlag
			return 0
		},
	},
	0x9E: {
		Mnemonic: "SBC_A_[HL]",
		Size:     1,
		Ticks:    []uint8{8},
		AddrMode: NONE,
		Operation: func(c *CPU) uint8 {
			val := c.bus.Read(c.Registers.GetHL())
			c.cpuCycles(1)
			a := c.Registers.A
			carryFlag := c.SetSbcFlags(a, val)
			c.Registers.A -= (val + carryFlag)
			return 0
		},
	},
	0x9F: {
		Mnemonic: "SBC_A_A",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) uint8 {
			a := c.Registers.A
			oldCarryFlag := c.Registers.GetFlag(common.CARRY_FLAG)
			carryFlag := c.SetSbcFlags(a, a)
			c.Registers.A -= (a - carryFlag)
			// set carryFlag back to original value as it should not be affected by this opcode
			c.Registers.SetFlag(common.SUBTRACTION_FLAG, oldCarryFlag)
			return 0
		},
	},
	0xA0: {
		Mnemonic: "AND_A_B",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) uint8 {
			c.Registers.A &= c.Registers.B
			c.SetAndFlags(c.Registers.A)
			return 0
		},
	},
	0xA1: {
		Mnemonic: "AND_A_C",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) uint8 {
			c.Registers.A &= c.Registers.C
			c.SetAndFlags(c.Registers.A)
			return 0
		},
	},
	0xA2: {
		Mnemonic: "AND_A_D",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) uint8 {
			c.Registers.A &= c.Registers.D
			c.SetAndFlags(c.Registers.A)
			return 0
		},
	},
	0xA3: {
		Mnemonic: "AND_A_E",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) uint8 {
			c.Registers.A &= c.Registers.E
			c.SetAndFlags(c.Registers.A)
			return 0
		},
	},
	0xA4: {
		Mnemonic: "AND_A_H",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) uint8 {
			c.Registers.A &= c.Registers.H
			c.SetAndFlags(c.Registers.A)
			return 0
		},
	},
	0xA5: {
		Mnemonic: "AND_A_L",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) uint8 {
			c.Registers.A &= c.Registers.L
			c.SetAndFlags(c.Registers.A)
			return 0
		},
	},
	0xA6: {
		Mnemonic: "AND_A_[HL]",
		Size:     1,
		Ticks:    []uint8{8},
		AddrMode: NONE,
		Operation: func(c *CPU) uint8 {
			val := c.bus.Read(c.Registers.GetHL())
			c.cpuCycles(1)
			c.Registers.A &= val
			c.SetAndFlags(c.Registers.A)
			return 0
		},
	},
	0xA7: {
		Mnemonic: "AND_A_A",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) uint8 {
			c.Registers.A &= c.Registers.A
			c.SetAndFlags(c.Registers.A)
			return 0
		},
	},
	0xA8: {
		Mnemonic: "XOR_A_B",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) uint8 {
			c.Registers.A ^= c.Registers.B
			c.SetXorFlags(c.Registers.A)
			return 0
		},
	},
	0xA9: {
		Mnemonic: "XOR_A_C",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) uint8 {
			c.Registers.A ^= c.Registers.C
			c.SetXorFlags(c.Registers.A)
			return 0
		},
	},
	0xAA: {
		Mnemonic: "XOR_A_D",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) uint8 {
			c.Registers.A ^= c.Registers.D
			c.SetXorFlags(c.Registers.A)
			return 0
		},
	},
	0xAB: {
		Mnemonic: "XOR_A_E",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) uint8 {
			c.Registers.A ^= c.Registers.E
			c.SetXorFlags(c.Registers.A)
			return 0
		},
	},
	0xAC: {
		Mnemonic: "XOR_A_H",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) uint8 {
			c.Registers.A ^= c.Registers.H
			c.SetXorFlags(c.Registers.A)
			return 0
		},
	},
	0xAD: {
		Mnemonic: "XOR_A_L",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) uint8 {
			c.Registers.A ^= c.Registers.L
			c.SetXorFlags(c.Registers.A)
			return 0
		},
	},
	0xAE: {
		Mnemonic: "XOR_A_[HL]",
		Size:     1,
		Ticks:    []uint8{8},
		AddrMode: NONE,
		Operation: func(c *CPU) uint8 {
			val := c.bus.Read(c.Registers.GetHL())
			c.cpuCycles(1)
			c.Registers.A ^= val
			c.SetXorFlags(c.Registers.A)
			return 0
		},
	},
	0xAF: {
		Mnemonic: "XOR_A_A",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) uint8 {
			c.Registers.A ^= c.Registers.A
			c.SetXorFlags(c.Registers.A)
			return 0
		},
	},
	0xB0: {
		Mnemonic: "OR_A_B",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) uint8 {
			c.Registers.A |= c.Registers.B
			c.SetXorFlags(c.Registers.A)
			return 0
		},
	},
	0xB1: {
		Mnemonic: "OR_A_C",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) uint8 {
			c.Registers.A |= c.Registers.C
			c.SetXorFlags(c.Registers.A)
			return 0
		},
	},
	0xB2: {
		Mnemonic: "OR_A_D",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) uint8 {
			c.Registers.A |= c.Registers.D
			c.SetXorFlags(c.Registers.A)
			return 0
		},
	},
	0xB3: {
		Mnemonic: "OR_A_E",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) uint8 {
			c.Registers.A |= c.Registers.E
			c.SetXorFlags(c.Registers.A)
			return 0
		},
	},
	0xB4: {
		Mnemonic: "OR_A_H",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) uint8 {
			c.Registers.A |= c.Registers.H
			c.SetXorFlags(c.Registers.A)
			return 0
		},
	},
	0xB5: {
		Mnemonic: "OR_A_L",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) uint8 {
			c.Registers.A |= c.Registers.L
			c.SetXorFlags(c.Registers.A)
			return 0
		},
	},
	0xB6: {
		Mnemonic: "OR_A_[HL]",
		Size:     1,
		Ticks:    []uint8{8},
		AddrMode: NONE,
		Operation: func(c *CPU) uint8 {
			val := c.bus.Read(c.Registers.GetHL())
			c.cpuCycles(1)
			c.Registers.A |= val
			c.SetXorFlags(c.Registers.A)
			return 0
		},
	},
	0xB7: {
		Mnemonic: "OR_A_A",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) uint8 {
			c.Registers.A |= c.Registers.A
			c.SetXorFlags(c.Registers.A)
			return 0
		},
	},
	0xB8: {
		Mnemonic: "CP_A_B",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) uint8 {
			c.SetCpFlags(c.Registers.A, c.Registers.B)
			return 0
		},
	},
	0xB9: {
		Mnemonic: "CP_A_C",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) uint8 {
			c.SetCpFlags(c.Registers.A, c.Registers.C)
			return 0
		},
	},
	0xBA: {
		Mnemonic: "CP_A_D",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) uint8 {
			c.SetCpFlags(c.Registers.A, c.Registers.D)
			return 0
		},
	},
	0xBB: {
		Mnemonic: "CP_A_E",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) uint8 {
			c.SetCpFlags(c.Registers.A, c.Registers.E)
			return 0
		},
	},
	0xBC: {
		Mnemonic: "CP_A_H",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) uint8 {
			c.SetCpFlags(c.Registers.A, c.Registers.H)
			return 0
		},
	},
	0xBD: {
		Mnemonic: "CP_A_L",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) uint8 {
			c.SetCpFlags(c.Registers.A, c.Registers.L)
			return 0
		},
	},
	0xBE: {
		Mnemonic: "CP_A_[HL]",
		Size:     1,
		Ticks:    []uint8{8},
		AddrMode: NONE,
		Operation: func(c *CPU) uint8 {
			val := c.bus.Read(c.Registers.GetHL())
			c.cpuCycles(1)
			c.SetCpFlags(c.Registers.A, val)
			return 0
		},
	},
	0xBF: {
		Mnemonic: "CP_A_A",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) uint8 {
			c.SetCpFlags(c.Registers.A, c.Registers.A)
			return 0
		},
	},
	0xC0: {
		Mnemonic: "RET_NZ",
		Size:     1,
		Ticks:    []uint8{20, 8},
		AddrMode: NONE,
		Operation: func(c *CPU) uint8 {
			if !c.Registers.GetFlag(common.ZERO_FLAG) {
				val := c.StackPop16()
				c.cpuCycles(1)
				c.Registers.PC = val
				c.cpuCycles(1)
				return 0
			}
			c.cpuCycles(1)
			return 1
		},
	},
	0xC1: {
		Mnemonic: "POP_BC",
		Size:     1,
		Ticks:    []uint8{12},
		AddrMode: NONE,
		Operation: func(c *CPU) uint8 {
			val := c.StackPop16()
			c.Registers.SetBC(val)
			return 0
		},
	},
	0xC2: {
		Mnemonic: "JP_NZ_A16",
		Size:     3,
		Ticks:    []uint8{16, 12},
		AddrMode: R_A16,
		Operation: func(c *CPU) uint8 {
			if !c.Registers.GetFlag(common.ZERO_FLAG) {
				c.Registers.PC = c.Fetched
				c.cpuCycles(1)
				return 0
			}
			return 1
		},
	},
	0xC3: {
		Mnemonic: "JP_A16",
		Size:     3,
		Ticks:    []uint8{12},
		AddrMode: R_A16,
		Operation: func(c *CPU) uint8 {
			c.Registers.PC = c.Fetched
			c.cpuCycles(1)
			return 0
		},
	},
	0xC4: {
		Mnemonic: "CALL_NZ_A16",
		Size:     3,
		Ticks:    []uint8{24, 12},
		AddrMode: R_A16,
		Operation: func(c *CPU) uint8 {
			if !c.Registers.GetFlag(common.ZERO_FLAG) {
				c.StackPush16(c.Registers.PC)
				c.Registers.PC = c.Fetched
				c.cpuCycles(1)
				return 0
			}
			return 1
		},
	},
	0xC5: {
		Mnemonic: "PUSH_BC",
		Size:     1,
		Ticks:    []uint8{16},
		AddrMode: NONE,
		Operation: func(c *CPU) uint8 {
			c.StackPush16(c.Registers.GetBC())
			c.cpuCycles(1)
			return 0
		},
	},
	0xC6: {
		Mnemonic: "ADD_A_N8",
		Size:     2,
		Ticks:    []uint8{8},
		AddrMode: R_N8,
		Operation: func(c *CPU) uint8 {
			a := c.Registers.A
			c.SetAddFlags(a, uint8(c.Fetched))
			c.Registers.A += uint8(c.Fetched)
			return 0
		},
	},
	0xC7: {
		Mnemonic: "RST_$00",
		Size:     3,
		Ticks:    []uint8{16},
		AddrMode: NONE,
		Operation: func(c *CPU) uint8 {
			c.StackPush16(c.Registers.PC)
			c.Registers.PC = 0x0000
			c.cpuCycles(1)
			return 0
		},
	},
	0xC8: {
		Mnemonic: "RET_Z",
		Size:     1,
		Ticks:    []uint8{20, 8},
		AddrMode: NONE,
		Operation: func(c *CPU) uint8 {
			if c.Registers.GetFlag(common.ZERO_FLAG) {
				val := c.StackPop16()
				c.cpuCycles(1)
				c.Registers.PC = val
				c.cpuCycles(1)
				return 0
			}
			c.cpuCycles(1)
			return 1
		},
	},
	0xC9: {
		Mnemonic: "RET",
		Size:     1,
		Ticks:    []uint8{16},
		AddrMode: NONE,
		Operation: func(c *CPU) uint8 {
			val := c.StackPop16()
			c.Registers.PC = val
			c.cpuCycles(1)
			return 0
		},
	},
	0xCA: {
		Mnemonic: "JP_Z_A16",
		Size:     3,
		Ticks:    []uint8{16, 12},
		AddrMode: R_A16,
		Operation: func(c *CPU) uint8 {
			if c.Registers.GetFlag(common.ZERO_FLAG) {
				c.Registers.PC = c.Fetched
				c.cpuCycles(1)
				return 0
			}
			return 1
		},
	},
	0xCB: {
		Mnemonic: "CB",
		Size:     3,
		Ticks:    []uint8{16, 12, 8},
		AddrMode: R_N8,
		Operation: func(c *CPU) uint8 {
			// 0000000010001101
			firstNibble := (c.Fetched >> 4) & 0x000F
			secondNibble := c.Fetched & 0x000F
			switch firstNibble {
			// rlc rrc
			case 0x0:
				if secondNibble < 0x8 {
					// rlc
					if secondNibble != 0x6 {
						reg := c.CBLookUp(uint8(secondNibble))
						*reg = c.SetCBRotateFlags(*reg, "L", false)
						return 2
					} else {
						val := c.bus.Read(c.Registers.GetHL())
						c.cpuCycles(1)
						c.bus.Write(c.Registers.GetHL(), c.SetCBRotateFlags(val, "L", false))
						c.cpuCycles(1)
						return 0
					}
				} else {
					// rrc
					if secondNibble != 0xE {
						reg := c.CBLookUp(uint8(secondNibble))
						*reg = c.SetCBRotateFlags(*reg, "R", false)
						return 2
					} else {
						val := c.bus.Read(c.Registers.GetHL())
						c.cpuCycles(1)
						c.bus.Write(c.Registers.GetHL(), c.SetCBRotateFlags(val, "R", false))
						c.cpuCycles(1)
						return 0
					}
				}
			// rl rr
			case 0x1:
				if secondNibble < 0x8 {
					// rl
					if secondNibble != 0x6 {
						reg := c.CBLookUp(uint8(secondNibble))
						*reg = c.SetCBRotateFlags(*reg, "L", true)
						return 2
					} else {
						val := c.bus.Read(c.Registers.GetHL())
						c.cpuCycles(1)
						c.bus.Write(c.Registers.GetHL(), c.SetCBRotateFlags(val, "L", true))
						c.cpuCycles(1)
						return 0
					}
				} else {
					// rr
					if secondNibble != 0xE {
						reg := c.CBLookUp(uint8(secondNibble))
						*reg = c.SetCBRotateFlags(*reg, "R", true)
						return 2
					} else {
						val := c.bus.Read(c.Registers.GetHL())
						c.cpuCycles(1)
						c.bus.Write(c.Registers.GetHL(), c.SetCBRotateFlags(val, "R", true))
						c.cpuCycles(1)
						return 0
					}
				}
			// sla sra
			case 0x2:
				if secondNibble < 0x8 {
					// sla
					if secondNibble != 0x6 {
						reg := c.CBLookUp(uint8(secondNibble))
						*reg = c.SetShiftFlags(*reg, "L", false)
						return 2
					} else {
						val := c.bus.Read(c.Registers.GetHL())
						c.cpuCycles(1)
						c.bus.Write(c.Registers.GetHL(), c.SetShiftFlags(val, "L", false))
						c.cpuCycles(1)
						return 0
					}
				} else {
					// sra
					if secondNibble != 0xE {
						reg := c.CBLookUp(uint8(secondNibble))
						*reg = c.SetShiftFlags(*reg, "R", false)
						return 2
					} else {
						val := c.bus.Read(c.Registers.GetHL())
						c.cpuCycles(1)
						c.bus.Write(c.Registers.GetHL(), c.SetShiftFlags(val, "R", false))
						c.cpuCycles(1)
						return 0
					}
				}
			// swap srl
			case 0x3:
				if secondNibble < 0x8 {
					// swap
					if secondNibble != 0x6 {
						reg := c.CBLookUp(uint8(secondNibble))
						*reg = c.SetSwapFlags(*reg)
						return 2
					} else {
						val := c.bus.Read(c.Registers.GetHL())
						c.cpuCycles(1)
						c.bus.Write(c.Registers.GetHL(), c.SetSwapFlags(val))
						c.cpuCycles(1)
						return 0
					}
				} else {
					// srl
					if secondNibble != 0xE {
						reg := c.CBLookUp(uint8(secondNibble))
						*reg = c.SetShiftFlags(*reg, "R", true)
						return 2
					} else {
						val := c.bus.Read(c.Registers.GetHL())
						c.cpuCycles(1)
						c.bus.Write(c.Registers.GetHL(), c.SetShiftFlags(val, "R", true))
						c.cpuCycles(1)
						return 0
					}
				}
			// BIT 0 BIT 1
			case 0x4:
				if secondNibble < 0x8 {
					// bit 0
					if secondNibble != 0x6 {
						val := c.CBLookUp(uint8(secondNibble))
						c.SetBitFlags(*val, 0)
						return 2
					} else {
						val := c.bus.Read(c.Registers.GetHL())
						c.cpuCycles(1)
						c.SetBitFlags(val, 0)
						return 1
					}
				} else {
					// bit 1
					if secondNibble != 0xE {
						val := c.CBLookUp(uint8(secondNibble))
						c.SetBitFlags(*val, 1)
						return 2
					} else {
						val := c.bus.Read(c.Registers.GetHL())
						c.cpuCycles(1)
						c.SetBitFlags(val, 1)
						return 1
					}
				}
			// BIT 2 BIT 3
			case 0x5:
				if secondNibble < 0x8 {
					// bit 2
					if secondNibble != 0x6 {
						val := c.CBLookUp(uint8(secondNibble))
						c.SetBitFlags(*val, 2)
						return 2
					} else {
						val := c.bus.Read(c.Registers.GetHL())
						c.cpuCycles(1)
						c.SetBitFlags(val, 2)
						return 1
					}
				} else {
					// bit 3
					if secondNibble != 0xE {
						val := c.CBLookUp(uint8(secondNibble))
						c.SetBitFlags(*val, 3)
						return 2
					} else {
						val := c.bus.Read(c.Registers.GetHL())
						c.cpuCycles(1)
						c.SetBitFlags(val, 3)
						return 1
					}
				}
			// BIT 4 BIT 5
			case 0x6:
				if secondNibble < 0x8 {
					// bit 4
					if secondNibble != 0x6 {
						val := c.CBLookUp(uint8(secondNibble))
						c.SetBitFlags(*val, 4)
						return 2
					} else {
						val := c.bus.Read(c.Registers.GetHL())
						c.cpuCycles(1)
						c.SetBitFlags(val, 4)
						return 1
					}
				} else {
					// bit 5
					if secondNibble != 0xE {
						val := c.CBLookUp(uint8(secondNibble))
						c.SetBitFlags(*val, 5)
						return 2
					} else {
						val := c.bus.Read(c.Registers.GetHL())
						c.cpuCycles(1)
						c.SetBitFlags(val, 5)
						return 1
					}
				}
			// BIT 6 BIT 7
			case 0x7:
				if secondNibble < 0x8 {
					// bit 6
					if secondNibble != 0x6 {
						val := c.CBLookUp(uint8(secondNibble))
						c.SetBitFlags(*val, 6)
						return 2
					} else {
						val := c.bus.Read(c.Registers.GetHL())
						c.cpuCycles(1)
						c.SetBitFlags(val, 6)
						return 1
					}
				} else {
					// bit 7
					if secondNibble != 0xE {
						val := c.CBLookUp(uint8(secondNibble))
						c.SetBitFlags(*val, 7)
						return 2
					} else {
						val := c.bus.Read(c.Registers.GetHL())
						c.cpuCycles(1)
						c.SetBitFlags(val, 7)
						return 1
					}
				}
			// RES 0 RES 1
			case 0x8:
				if secondNibble < 0x8 {
					// RES 0
					if secondNibble != 0x6 {
						val := c.CBLookUp(uint8(secondNibble))
						*val = c.SetBit(*val, 0, false)
						return 2
					} else {
						val := c.bus.Read(c.Registers.GetHL())
						c.cpuCycles(1)
						val = c.SetBit(val, 0, false)
						c.bus.Write(c.Registers.GetHL(), val)
						c.cpuCycles(1)
						return 0
					}
				} else {
					// RES 1
					if secondNibble != 0xE {
						val := c.CBLookUp(uint8(secondNibble))
						*val = c.SetBit(*val, 1, false)
						return 2
					} else {
						val := c.bus.Read(c.Registers.GetHL())
						c.cpuCycles(1)
						val = c.SetBit(val, 1, false)
						c.bus.Write(c.Registers.GetHL(), val)
						c.cpuCycles(1)
						return 0
					}
				}
			// RES 2 RES 3
			case 0x9:
				if secondNibble < 0x8 {
					// RES 2
					if secondNibble != 0x6 {
						val := c.CBLookUp(uint8(secondNibble))
						*val = c.SetBit(*val, 2, false)
						return 2
					} else {
						val := c.bus.Read(c.Registers.GetHL())
						c.cpuCycles(1)
						val = c.SetBit(val, 2, false)
						c.bus.Write(c.Registers.GetHL(), val)
						c.cpuCycles(1)
						return 0
					}
				} else {
					// RES 3
					if secondNibble != 0xE {
						val := c.CBLookUp(uint8(secondNibble))
						*val = c.SetBit(*val, 3, false)
						return 2
					} else {
						val := c.bus.Read(c.Registers.GetHL())
						c.cpuCycles(1)
						val = c.SetBit(val, 3, false)
						c.bus.Write(c.Registers.GetHL(), val)
						c.cpuCycles(1)
						return 0
					}
				}
			// RES 4 RES 5
			case 0xA:
				if secondNibble < 0x8 {
					// RES 4
					if secondNibble != 0x6 {
						val := c.CBLookUp(uint8(secondNibble))
						*val = c.SetBit(*val, 4, false)
						return 2
					} else {
						val := c.bus.Read(c.Registers.GetHL())
						c.cpuCycles(1)
						val = c.SetBit(val, 4, false)
						c.bus.Write(c.Registers.GetHL(), val)
						c.cpuCycles(1)
						return 0
					}
				} else {
					// RES 5
					if secondNibble != 0xE {
						val := c.CBLookUp(uint8(secondNibble))
						*val = c.SetBit(*val, 5, false)
						return 2
					} else {
						val := c.bus.Read(c.Registers.GetHL())
						c.cpuCycles(1)
						val = c.SetBit(val, 5, false)
						c.bus.Write(c.Registers.GetHL(), val)
						c.cpuCycles(1)
						return 0
					}
				}
			// RES 6 RES 7
			case 0xB:
				if secondNibble < 0x8 {
					// RES 6
					if secondNibble != 0x6 {
						val := c.CBLookUp(uint8(secondNibble))
						*val = c.SetBit(*val, 6, false)
						return 2
					} else {
						val := c.bus.Read(c.Registers.GetHL())
						c.cpuCycles(1)
						val = c.SetBit(val, 6, false)
						c.bus.Write(c.Registers.GetHL(), val)
						c.cpuCycles(1)
						return 0
					}
				} else {
					// RES 7
					if secondNibble != 0xE {
						val := c.CBLookUp(uint8(secondNibble))
						*val = c.SetBit(*val, 7, false)
						return 2
					} else {
						val := c.bus.Read(c.Registers.GetHL())
						c.cpuCycles(1)
						val = c.SetBit(val, 7, false)
						c.bus.Write(c.Registers.GetHL(), val)
						c.cpuCycles(1)
						return 0
					}
				}
			// SET 0 SET 1
			case 0xC:
				if secondNibble < 0x8 {
					// SET 0
					if secondNibble != 0x6 {
						val := c.CBLookUp(uint8(secondNibble))
						*val = c.SetBit(*val, 0, true)
						return 2
					} else {
						val := c.bus.Read(c.Registers.GetHL())
						c.cpuCycles(1)
						val = c.SetBit(val, 0, true)
						c.bus.Write(c.Registers.GetHL(), val)
						c.cpuCycles(1)
						return 0
					}
				} else {
					// SET 1
					if secondNibble != 0xE {
						val := c.CBLookUp(uint8(secondNibble))
						*val = c.SetBit(*val, 1, true)
						return 2
					} else {
						val := c.bus.Read(c.Registers.GetHL())
						c.cpuCycles(1)
						val = c.SetBit(val, 1, true)
						c.bus.Write(c.Registers.GetHL(), val)
						c.cpuCycles(1)
						return 0
					}
				}
			// SET 2 SET 3
			case 0xD:
				if secondNibble < 0x8 {
					// SET 2
					if secondNibble != 0x6 {
						val := c.CBLookUp(uint8(secondNibble))
						*val = c.SetBit(*val, 2, true)
						return 2
					} else {
						val := c.bus.Read(c.Registers.GetHL())
						c.cpuCycles(1)
						val = c.SetBit(val, 2, true)
						c.bus.Write(c.Registers.GetHL(), val)
						c.cpuCycles(1)
						return 0
					}
				} else {
					// SET 3
					if secondNibble != 0xE {
						val := c.CBLookUp(uint8(secondNibble))
						*val = c.SetBit(*val, 3, true)
						return 2
					} else {
						val := c.bus.Read(c.Registers.GetHL())
						c.cpuCycles(1)
						val = c.SetBit(val, 3, true)
						c.bus.Write(c.Registers.GetHL(), val)
						c.cpuCycles(1)
						return 0
					}
				}
			// SET 4 SET 5
			case 0xE:
				if secondNibble < 0x8 {
					// SET 4
					if secondNibble != 0x6 {
						val := c.CBLookUp(uint8(secondNibble))
						*val = c.SetBit(*val, 4, true)
						return 2
					} else {
						val := c.bus.Read(c.Registers.GetHL())
						c.cpuCycles(1)
						val = c.SetBit(val, 4, true)
						c.bus.Write(c.Registers.GetHL(), val)
						c.cpuCycles(1)
						return 0
					}
				} else {
					// SET 5
					if secondNibble != 0xE {
						val := c.CBLookUp(uint8(secondNibble))
						*val = c.SetBit(*val, 5, true)
						return 2
					} else {
						val := c.bus.Read(c.Registers.GetHL())
						c.cpuCycles(1)
						val = c.SetBit(val, 5, true)
						c.bus.Write(c.Registers.GetHL(), val)
						c.cpuCycles(1)
						return 0
					}
				}
			// SET 6 SET 7
			case 0xF:
				if secondNibble < 0x8 {
					// SET 6
					if secondNibble != 0x6 {
						val := c.CBLookUp(uint8(secondNibble))
						*val = c.SetBit(*val, 6, true)
						return 2
					} else {
						val := c.bus.Read(c.Registers.GetHL())
						c.cpuCycles(1)
						val = c.SetBit(val, 6, true)
						c.bus.Write(c.Registers.GetHL(), val)
						c.cpuCycles(1)
						return 0
					}
				} else {
					// SET 7
					if secondNibble != 0xE {
						val := c.CBLookUp(uint8(secondNibble))
						*val = c.SetBit(*val, 7, true)
						return 2
					} else {
						val := c.bus.Read(c.Registers.GetHL())
						c.cpuCycles(1)
						val = c.SetBit(val, 7, true)
						c.bus.Write(c.Registers.GetHL(), val)
						c.cpuCycles(1)
						return 0
					}
				}
			}
			return 2
		},
	},
	0xCC: {
		Mnemonic: "CALL_Z_A16",
		Size:     3,
		Ticks:    []uint8{24, 12},
		AddrMode: R_A16,
		Operation: func(c *CPU) uint8 {
			if c.Registers.GetFlag(common.ZERO_FLAG) {
				c.StackPush16(c.Registers.PC)
				c.Registers.PC = c.Fetched
				c.cpuCycles(1)
				return 0
			}
			return 1
		},
	},
	0xCD: {
		Mnemonic: "CALL_A16",
		Size:     3,
		Ticks:    []uint8{24},
		AddrMode: A16_R,
		Operation: func(c *CPU) uint8 {
			c.StackPush16(c.Registers.PC)
			c.Registers.PC = c.Fetched
			c.cpuCycles(1)
			return 0
		},
	},
	0xCE: {
		Mnemonic: "ADC_A_N8",
		Size:     2,
		Ticks:    []uint8{8},
		AddrMode: R_N8,
		Operation: func(c *CPU) uint8 {
			a := c.Registers.A
			carryFlag := c.SetAdcFlags(a, uint8(c.Fetched))
			c.Registers.A = (a + uint8(c.Fetched) + carryFlag) & 0x00FF
			return 0
		},
	},
	0xCF: {
		Mnemonic: "RST_$08",
		Size:     3,
		Ticks:    []uint8{16},
		AddrMode: NONE,
		Operation: func(c *CPU) uint8 {
			c.StackPush16(c.Registers.PC)
			c.Registers.PC = 0x0008
			c.cpuCycles(1)
			return 0
		},
	},
	0xD0: {
		Mnemonic: "RET_NC",
		Size:     1,
		Ticks:    []uint8{20, 8},
		AddrMode: NONE,
		Operation: func(c *CPU) uint8 {
			if !c.Registers.GetFlag(common.CARRY_FLAG) {
				val := c.StackPop16()
				c.cpuCycles(1)
				c.Registers.PC = (val)
				c.cpuCycles(1)
				return 0
			}
			c.cpuCycles(1)
			return 1
		},
	},
	0xD1: {
		Mnemonic: "POP_DE",
		Size:     1,
		Ticks:    []uint8{12},
		AddrMode: NONE,
		Operation: func(c *CPU) uint8 {
			val := c.StackPop16()
			c.Registers.SetDE(val)
			return 0
		},
	},
	0xD2: {
		Mnemonic: "JP_NC_A16",
		Size:     3,
		Ticks:    []uint8{16, 12},
		AddrMode: R_A16,
		Operation: func(c *CPU) uint8 {
			if !c.Registers.GetFlag(common.CARRY_FLAG) {
				c.Registers.PC = c.Fetched
				c.cpuCycles(1)
				return 0
			}
			return 1
		},
	},
	0xD3: DASH,
	0xD4: {
		Mnemonic: "CALL_NC_A16",
		Size:     3,
		Ticks:    []uint8{24, 12},
		AddrMode: R_A16,
		Operation: func(c *CPU) uint8 {
			if !c.Registers.GetFlag(common.CARRY_FLAG) {
				c.StackPush16(c.Registers.PC)
				c.Registers.PC = c.Fetched
				c.cpuCycles(1)
				return 0
			}
			return 1
		},
	},
	0xD5: {
		Mnemonic: "PUSH_DE",
		Size:     1,
		Ticks:    []uint8{16},
		AddrMode: NONE,
		Operation: func(c *CPU) uint8 {
			c.StackPush16(c.Registers.GetDE())
			c.cpuCycles(1)
			return 0
		},
	},
	0xD6: {
		Mnemonic: "SUB_A_N8",
		Size:     1,
		Ticks:    []uint8{8},
		AddrMode: R_N8,
		Operation: func(c *CPU) uint8 {
			a := c.Registers.A
			c.SetSubFlags(a, uint8(c.Fetched))
			c.Registers.A -= uint8(c.Fetched)
			return 0
		},
	},
	0xD7: {
		Mnemonic: "RST_$10",
		Size:     3,
		Ticks:    []uint8{16},
		AddrMode: NONE,
		Operation: func(c *CPU) uint8 {
			c.StackPush16(c.Registers.PC)
			c.Registers.PC = 0x0010
			c.cpuCycles(1)
			return 0
		},
	},
	0xD8: {
		Mnemonic: "RET_C",
		Size:     1,
		Ticks:    []uint8{20, 8},
		AddrMode: NONE,
		Operation: func(c *CPU) uint8 {
			if c.Registers.GetFlag(common.CARRY_FLAG) {
				val := c.StackPop16()
				c.cpuCycles(1)
				c.Registers.PC = val
				c.cpuCycles(1)
				return 0
			}
			c.cpuCycles(1)
			return 1
		},
	},
	0xD9: {
		Mnemonic: "RETI",
		Size:     1,
		Ticks:    []uint8{16},
		AddrMode: NONE,
		Operation: func(c *CPU) uint8 {
			val := c.StackPop16()
			c.Registers.PC = val
			c.cpuCycles(1)
			c.Registers.IME = true
			return 0
		},
	},
	0xDA: {
		Mnemonic: "JP_C_A16",
		Size:     3,
		Ticks:    []uint8{16, 12},
		AddrMode: R_A16,
		Operation: func(c *CPU) uint8 {
			if c.Registers.GetFlag(common.CARRY_FLAG) {
				c.Registers.PC = c.Fetched
				c.cpuCycles(1)
				return 0
			}
			return 1
		},
	},
	0xDB: DASH,
	0xDC: {
		Mnemonic: "CALL_C_A16",
		Size:     3,
		Ticks:    []uint8{24, 12},
		AddrMode: R_A16,
		Operation: func(c *CPU) uint8 {
			if c.Registers.GetFlag(common.CARRY_FLAG) {
				c.StackPush16(c.Registers.PC)
				c.Registers.PC = c.Fetched
				c.cpuCycles(1)
				return 0
			}
			return 1
		},
	},
	0xDD: DASH,
	0xDE: {
		Mnemonic: "SBC_A_N8",
		Size:     1,
		Ticks:    []uint8{8},
		AddrMode: R_N8,
		Operation: func(c *CPU) uint8 {
			a := c.Registers.A
			carryFlag := c.SetSbcFlags(a, uint8(c.Fetched))
			c.Registers.A = a - uint8(c.Fetched) - carryFlag
			return 0
		},
	},
	0xDF: {
		Mnemonic: "RST_$18",
		Size:     3,
		Ticks:    []uint8{16},
		AddrMode: NONE,
		Operation: func(c *CPU) uint8 {
			c.StackPush16(c.Registers.PC)
			c.Registers.PC = 0x0018
			c.cpuCycles(1)
			return 0
		},
	},
	0xE0: {
		Mnemonic: "LDH_[A8]_A",
		Size:     2,
		AddrMode: A8_A,
		Ticks:    []uint8{12},
		Operation: func(c *CPU) uint8 {
			c.bus.Write(c.Fetched, c.Registers.A)
			c.cpuCycles(1)
			return 0
		},
	},
	0xE1: {
		Mnemonic: "POP_HL",
		Size:     1,
		Ticks:    []uint8{12},
		AddrMode: NONE,
		Operation: func(c *CPU) uint8 {
			val := c.StackPop16()
			c.Registers.SetHL(val)
			return 0
		},
	},
	0xE2: {
		Mnemonic: "LDH_[C]_A",
		Size:     1,
		AddrMode: NONE,
		Ticks:    []uint8{8},
		Operation: func(c *CPU) uint8 {
			c.bus.Write(uint16(c.Registers.C)+0xFF00, c.Registers.A)
			c.cpuCycles(1)
			return 0
		},
	},
	0xE3: DASH,
	0xE4: DASH,
	0xE5: {
		Mnemonic: "PUSH_HL",
		Size:     1,
		Ticks:    []uint8{16},
		AddrMode: NONE,
		Operation: func(c *CPU) uint8 {
			c.StackPush16(c.Registers.GetHL())
			c.cpuCycles(1)
			return 0
		},
	},
	0xE6: {
		Mnemonic: "AND_A_N8",
		Size:     1,
		Ticks:    []uint8{8},
		AddrMode: R_N8,
		Operation: func(c *CPU) uint8 {
			c.Registers.A &= uint8(c.Fetched)
			c.SetAndFlags(c.Registers.A)
			return 0
		},
	},
	0xE7: {
		Mnemonic: "RST_$20",
		Size:     3,
		Ticks:    []uint8{16},
		AddrMode: NONE,
		Operation: func(c *CPU) uint8 {
			c.StackPush16(c.Registers.PC)
			c.Registers.PC = 0x0020
			c.cpuCycles(1)
			return 0
		},
	},
	0xE8: {
		Mnemonic: "ADD_SP_E8",
		Size:     2,
		Ticks:    []uint8{16},
		AddrMode: E8,
		Operation: func(c *CPU) uint8 {
			sp := c.Registers.SP
			c.SetAddFlags(uint8(sp&0x00FF), uint8(c.RelAddr))
			c.Registers.SetFlag(common.ZERO_FLAG, false)
			c.cpuCycles(1)
			c.Registers.SP = uint16(int16(sp) + int16(c.RelAddr))
			c.cpuCycles(1)
			return 0
		},
	},
	0xE9: {
		Mnemonic: "JP_HL",
		Size:     3,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) uint8 {
			c.Registers.PC = c.Registers.GetHL()
			return 0
		},
	},
	0xEA: {
		Mnemonic: "LD_[A16]_A",
		Size:     3,
		Ticks:    []uint8{16},
		AddrMode: A16_R,
		Operation: func(c *CPU) uint8 {
			c.bus.Write(c.Fetched, c.Registers.A)
			c.cpuCycles(1)
			return 0
		},
	},
	0xEB: DASH,
	0xEC: DASH,
	0xED: DASH,
	0xEE: {
		Mnemonic: "XOR_A_N8",
		Size:     1,
		Ticks:    []uint8{8},
		AddrMode: R_N8,
		Operation: func(c *CPU) uint8 {
			c.Registers.A ^= uint8(c.Fetched)
			c.SetXorFlags(c.Registers.A)
			return 0
		},
	},
	0xEF: {
		Mnemonic: "RST_$EF",
		Size:     3,
		Ticks:    []uint8{16},
		AddrMode: NONE,
		Operation: func(c *CPU) uint8 {
			c.StackPush16(c.Registers.PC)
			c.Registers.PC = 0x00EF
			c.cpuCycles(1)
			return 0
		},
	},
	0xF0: {
		Mnemonic: "LDH_A_[A8]",
		Size:     2,
		Ticks:    []uint8{12},
		AddrMode: A_A8,
		Operation: func(c *CPU) uint8 {
			c.Registers.A = c.bus.Read(c.Fetched)
			c.cpuCycles(1)
			return 0
		},
	},
	0xF1: {
		Mnemonic: "POP_AF",
		Size:     1,
		Ticks:    []uint8{12},
		AddrMode: NONE,
		Operation: func(c *CPU) uint8 {
			val := c.StackPop16()
			c.Registers.SetAF(val)
			return 0
		},
	},
	// 0xF2: {
	// 	Mnemonic: "LDH_A_[C]",
	// 	Size:     1,
	// 	AddrMode: NONE,
	// 	Ticks:    []uint8{8},
	// 	Operation: func(c *CPU) {
	// 		val := c.bus.Read(uint16(c.registers.GetReg(registers.C)) + 0xFF00)
	// 		c.registers.SetReg(registers.A, val)
	// 		c.cpuCycles(1)
	// 	},
	// },
	0xF3: {
		Mnemonic: "DI",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) uint8 {
			c.Registers.IME = false
			return 0
		},
	},
	// 0xF4: DASH,
	0xF5: {
		Mnemonic: "PUSH_AF",
		Size:     1,
		Ticks:    []uint8{16},
		AddrMode: NONE,
		Operation: func(c *CPU) uint8 {
			c.StackPush16(c.Registers.GetAF())
			c.cpuCycles(1)
			return 0
		},
	},
	0xF6: {
		Mnemonic: "OR_A_N8",
		Size:     1,
		Ticks:    []uint8{8},
		AddrMode: R_N8,
		Operation: func(c *CPU) uint8 {
			c.Registers.A |= uint8(c.Fetched)
			c.SetXorFlags(c.Registers.A)
			return 0
		},
	},
	0xF7: {
		Mnemonic: "RST_$30",
		Size:     3,
		Ticks:    []uint8{16},
		AddrMode: NONE,
		Operation: func(c *CPU) uint8 {
			c.StackPush16(c.Registers.PC)
			c.Registers.PC=0x0030
			c.cpuCycles(1)
			return 0
		},
	},
	0xF8: {
		Mnemonic: "LD_HL_SP+E8",
		Size:     2,
		Ticks:    []uint8{12},
		AddrMode: E8,
		Operation: func(c *CPU) uint8 {
			c.Registers.SetHL(uint16(int16(c.Registers.SP) + int16(c.RelAddr)))
			c.cpuCycles(1)
			return 0
		},
	},
	0xF9: {
		Mnemonic: "LD_SP_HL",
		Size:     1,
		Ticks:    []uint8{8},
		AddrMode: NONE,
		Operation: func(c *CPU) uint8 {
			c.Registers.SP = c.Registers.GetHL()
			c.cpuCycles(1)
			return 0
		},
	},
	0xFA: {
		Mnemonic: "LD_A_[A16]",
		Size:     3,
		Ticks:    []uint8{16},
		AddrMode: A16_R,
		Operation: func(c *CPU) uint8 {
			c.Registers.A = c.bus.Read(c.Fetched)
			c.cpuCycles(1)
			return 0
		},
	},
	0xFB: {
		Mnemonic: "EI",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) uint8 {
			c.EnablingIME = true
			return 0
		},
	},
	0xFC: DASH,
	0xFD: DASH,
	0xFE: {
		Mnemonic: "CP_A_N8",
		Size:     1,
		Ticks:    []uint8{8},
		AddrMode: R_N8,
		Operation: func(c *CPU) uint8 {
			c.SetCpFlags(c.Registers.A, uint8(c.Fetched))
			return 0
		},
	},
	0xFF: {
		Mnemonic: "RST_$38",
		Size:     3,
		Ticks:    []uint8{16},
		AddrMode: NONE,
		Operation: func(c *CPU) uint8 {
			c.StackPush16(c.Registers.PC)
			c.Registers.PC=0x0038
			c.cpuCycles(1)
			return 0
		},
	},
}

var DASH = Instruction{
	Mnemonic: "-",
	Size:     1,
	Ticks:    []uint8{0},
	AddrMode: NONE,
	Operation: func(c *CPU) uint8 {
		return 0
	},
}

var ticks = [0x100][]uint8{
	0x00: {4},
	0x01: {12},
	0x02: {8},
	0x03: {8},
	0x04: {4},
	0x05: {4},
	0x06: {8},
	0x07: {4},
	0x08: {20},
	0x09: {8},
	0x0A: {8},
	0x0B: {8},
	0x0C: {4},
	0x0D: {4},
	0x0E: {8},
	0x0F: {4},
	0x10: {4},
	0x11: {12},
	0x12: {8},
	0x13: {8},
	0x14: {4},
	0x15: {4},
	0x16: {8},
	0x17: {4},
	0x18: {12},
	0x19: {8},
	0x1A: {8},
	0x1B: {8},
	0x1C: {4},
	0x1D: {4},
	0x1E: {8},
	0x1F: {4},
	0x20: {12, 8},
	0x21: {12},
	0x22: {8},
	0x23: {8},
	0x24: {4},
	0x25: {4},
	0x26: {8},
	0x27: {4},
	0x28: {12, 8},
	0x29: {8},
	0x2A: {8},
	0x2B: {8},
	0x2C: {4},
	0x2D: {4},
	0x2E: {8},
	0x2F: {4},
	0x30: {12, 8},
	0x31: {12},
	0x32: {8},
	0x33: {8},
	0x34: {12},
	0x35: {12},
	0x36: {12},
	0x37: {4},
	0x38: {12, 8},
	0x39: {8},
	0x3A: {8},
	0x3B: {8},
	0x3C: {4},
	0x3D: {4},
	0x3E: {8},
	0x3F: {4},
	0x40: {4},
	0x41: {4},
	0x42: {4},
	0x43: {4},
	0x44: {4},
	0x45: {4},
	0x46: {8},
	0x47: {4},
	0x48: {4},
	0x49: {4},
	0x4A: {4},
	0x4B: {4},
	0x4C: {4},
	0x4D: {4},
	0x4E: {8},
	0x4F: {4},
	0x50: {4},
	0x51: {4},
	0x52: {4},
	0x53: {4},
	0x54: {4},
	0x55: {4},
	0x56: {8},
	0x57: {4},
	0x58: {4},
	0x59: {4},
	0x5A: {4},
	0x5B: {4},
	0x5C: {4},
	0x5D: {4},
	0x5E: {8},
	0x5F: {4},
	0x60: {4},
	0x61: {4},
	0x62: {4},
	0x63: {4},
	0x64: {4},
	0x65: {4},
	0x66: {8},
	0x67: {4},
	0x68: {4},
	0x69: {4},
	0x6A: {4},
	0x6B: {4},
	0x6C: {4},
	0x6D: {4},
	0x6E: {8},
	0x6F: {4},
	0x70: {8},
	0x71: {8},
	0x72: {8},
	0x73: {8},
	0x74: {8},
	0x75: {8},
	0x76: {4},
	0x77: {8},
	0x78: {4},
	0x79: {4},
	0x7A: {4},
	0x7B: {4},
	0x7C: {4},
	0x7D: {4},
	0x7E: {8},
	0x7F: {4},
	0x80: {4},
	0x81: {4},
	0x82: {4},
	0x83: {4},
	0x84: {4},
	0x85: {4},
	0x86: {8},
	0x87: {4},
	0x88: {4},
	0x89: {4},
	0x8A: {4},
	0x8B: {4},
	0x8C: {4},
	0x8D: {4},
	0x8E: {8},
	0x8F: {4},
	0x90: {4},
	0x91: {4},
	0x92: {4},
	0x93: {4},
	0x94: {4},
	0x95: {4},
	0x96: {8},
	0x97: {4},
	0x98: {4},
	0x99: {4},
	0x9A: {4},
	0x9B: {4},
	0x9C: {4},
	0x9D: {4},
	0x9E: {8},
	0x9F: {4},
	0xA0: {4},
	0xA1: {4},
	0xA2: {4},
	0xA3: {4},
	0xA4: {4},
	0xA5: {4},
	0xA6: {8},
	0xA7: {4},
	0xA8: {4},
	0xA9: {4},
	0xAA: {4},
	0xAB: {4},
	0xAC: {4},
	0xAD: {4},
	0xAE: {8},
	0xAF: {4},
	0xB0: {4},
	0xB1: {4},
	0xB2: {4},
	0xB3: {4},
	0xB4: {4},
	0xB5: {4},
	0xB6: {8},
	0xB7: {4},
	0xB8: {4},
	0xB9: {4},
	0xBA: {4},
	0xBB: {4},
	0xBC: {4},
	0xBD: {4},
	0xBE: {8},
	0xBF: {4},
	0xC0: {20, 8},
	0xC1: {12},
	0xC2: {16, 12},
	0xC3: {16},
	0xC4: {24, 12},
	0xC5: {16},
	0xC6: {8},
	0xC7: {16},
	0xC8: {20, 8},
	0xC9: {16},
	0xCA: {16, 12},
	0xCB: {16, 12, 8},
	0xCC: {24, 12},
	0xCD: {24},
	0xCE: {8},
	0xCF: {16},
	0xD0: {20, 8},
	0xD1: {12},
	0xD2: {16, 12},
	0xD3: {4},
	0xD4: {24, 12},
	0xD5: {16},
	0xD6: {8},
	0xD7: {16},
	0xD8: {20, 8},
	0xD9: {16},
	0xDA: {16, 12},
	0xDB: {4},
	0xDC: {24, 12},
	0xDD: {4},
	0xDE: {8},
	0xDF: {16},
	0xE0: {12},
	0xE1: {12},
	0xE2: {8},
	0xE3: {4},
	0xE4: {4},
	0xE5: {16},
	0xE6: {8},
	0xE7: {16},
	0xE8: {16},
	0xE9: {4},
	0xEA: {16},
	0xEB: {4},
	0xEC: {4},
	0xED: {4},
	0xEE: {8},
	0xEF: {16},
	0xF0: {12},
	0xF1: {12},
	0xF2: {8},
	0xF3: {4},
	0xF4: {4},
	0xF5: {16},
	0xF6: {8},
	0xF7: {16},
	0xF8: {12},
	0xF9: {8},
	0xFA: {16},
	0xFB: {4},
	0xFC: {4},
	0xFD: {4},
	0xFE: {8},
	0xFF: {16},
}

func (c *CPU) CBLookUp(highNibble uint8) *uint8 {
	switch highNibble {
	case 0x0, 0x8:
		return &c.Registers.B
	case 0x1, 0x9:
		return &c.Registers.C
	case 0x2, 0xA:
		return &c.Registers.D
	case 0x3, 0xB:
		return &c.Registers.E
	case 0x4, 0xC:
		return &c.Registers.H
	case 0x5, 0xD:
		return &c.Registers.L
	default:
		return &c.Registers.A
	}
}
