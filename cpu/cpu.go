package cpu

import (
	"fmt"
	"goboy/cpu/registers"
	"os"
)

type CPU struct {
	Registers *registers.Registers
	CurInst   *Instruction

	Fetched  uint16
	DestAddr uint16
	RelAddr  int8
	Ticks    uint64
	Halted   bool
	Read     func(uint16) uint8
	Write    func(uint16, uint8)
}

func NewCPU() *CPU {
	registers := new(registers.Registers)
	return &CPU{
		Registers: registers,
	}
}

func (c *CPU) SetReadWrite(Read func(uint16) uint8, Write func(uint16, uint8)) {
	c.Read = Read
	c.Write = Write
}

func (c *CPU) Init() {
	c.Registers.A = 0x0011
	c.Registers.SetFlag(registers.ZERO_FLAG, true)
	c.Registers.B = 0x0000
	c.Registers.C = 0x0000
	c.Registers.D = 0x00FF
	c.Registers.E = 0x0056
	c.Registers.H = 0x0000
	c.Registers.L = 0x000D
	c.Registers.PC = 0x0100
	c.Registers.SP = 0xFFFE
}

func (c *CPU) StackPush(value uint8) {
	c.Registers.SP--
	c.Write(c.Registers.SP, value)
}

func (c *CPU) StackPush16(value uint16) {
	// push hi
	c.StackPush(uint8(value<<8) & 0x00FF)
	// push lo
	c.StackPush(uint8(value) & 0x00FF)
}

func (c *CPU) StackPop() uint8 {
	poppedValue := c.Read(c.Registers.SP)
	c.Registers.SP++
	return poppedValue
}

func (c *CPU) StackPop16() uint16 {
	// pop lo
	lo := uint16(c.StackPop())
	// pop hi
	hi := uint16(c.StackPop())
	return (hi << 8) | lo
}

func (c *CPU) execute() {
	pc := c.Registers.PC
	opcode := c.Read(pc)
	c.cpuCycles(1)
	if Instructions[opcode].Operation == nil {
		fmt.Printf("opcode: %04x not implemented\n", opcode)
		fmt.Printf("%02x 02%d 02%d\n", opcode, c.Read(c.Registers.PC+1), c.Read(c.Registers.PC+2))
		os.Exit(-1)
	}

	c.process(opcode)
}

func (c *CPU) process(opcode uint8) {
	c.CurInst = &Instructions[opcode]
	fmt.Printf("%-10s \t %02x %02x %02x ",
		c.CurInst.Mnemonic,
		opcode,
		c.Read(c.Registers.PC+1),
		c.Read(c.Registers.PC+2))
	c.Registers.PC++
	c.CurInst.AddrMode(c)
	c.CurInst.Operation(c)
	fmt.Printf("AF: 0b%016b BC: 0x%04x DE: 0x%04x HL: 0x%04x PC: 0x%04x SP: 0x%04x Ticks: %d\n",
		c.Registers.GetAF(),
		c.Registers.GetBC(),
		c.Registers.GetDE(),
		c.Registers.GetHL(),
		c.Registers.PC,
		c.Registers.SP,
		c.Ticks)
}

func (c *CPU) Step() bool {
	if !c.Halted {
		c.execute()
	}

	return true
}

type Operation func(c *CPU)
type AddrMode func(c *CPU)

type Condition uint8

const (
	C_Z Condition = iota
	C_NZ
	C_H
	C_C
	C_NC
	C_NONE
)

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
	return
}

// 16 bit address
func R_A16(c *CPU) {
	// grab low and hi byte from adddress pc and pc +1
	lo := c.Read(c.Registers.PC)
	c.cpuCycles(1)
	hi := c.Read(c.Registers.PC + 1)
	c.cpuCycles(1)
	c.Registers.PC += 2
	c.Fetched = (uint16(hi) << 8) | uint16(lo)
}

func A16_R(c *CPU) {
	// grab low and hi byte from adddress pc and pc +1
	lo := c.Read(c.Registers.PC)
	c.cpuCycles(1)
	hi := c.Read(c.Registers.PC + 1)
	c.cpuCycles(1)
	c.Registers.PC += 2
	c.Fetched = (uint16(hi) << 8) | uint16(lo)
}

func E8(c *CPU) {
	c.RelAddr = int8(c.Read(c.Registers.PC))
}

// 8 bit immediate data
func R_N8(c *CPU) {
	lo := c.Read(c.Registers.PC)
	c.cpuCycles(1)
	c.Registers.PC += 1
	c.Fetched = uint16(lo)
}

func A8_A(c *CPU) {
	lo := uint16(c.Read(c.Registers.PC)) + 0xFF00
	c.cpuCycles(1)
	c.Registers.PC += 1
	c.Fetched = lo
}

func A_A8(c *CPU) {
	lo := uint16(c.Read(c.Registers.PC)) + 0xFF00
	c.cpuCycles(1)
	c.Registers.PC += 1
	c.Fetched = lo
}

func HalfCarrySub(a uint8, b uint8) bool {
	// a = 00010000 b = 00000001
	// 00010000
	//-00000001
	//=00001111

	// a & 0x0f = 00000000
	// b & 0x0f = 00000001
	//          -
	//            11111111 // overflow
	// 11111111 & 00010000 = 00010000 // the 5th bit was flipped
	// 00010000 == 00010000
	// so we have a half carry
	if ((a&0x0f)-(b&0x0f))&0x10 == 0x10 {
		return true
	}
	return false
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
	if ((a&0x0f)+(b&0x0f))&0x10 == 0x10 {
		return true
	}
	return false
}

func (c *CPU) SetDecFlags(registerVal uint8) {
	c.Registers.SetFlag(registers.SUBTRACTION_FLAG, true)
	if registerVal-1 == 0 {
		c.Registers.SetFlag(registers.ZERO_FLAG, true)
	}
	if HalfCarrySub(registerVal, registerVal-1) {
		c.Registers.SetFlag(registers.HALF_CARRY_FLAG, true)
	}
}

func (c *CPU) SetIncFlags(registerVal uint8) {
	c.Registers.SetFlag(registers.SUBTRACTION_FLAG, false)
	if registerVal+1 == 0 {
		c.Registers.SetFlag(registers.ZERO_FLAG, true)
	}
	if HalfCarryAdd(registerVal, registerVal+1) {
		c.Registers.SetFlag(registers.HALF_CARRY_FLAG, true)
	}
}

func (c *CPU) SetRotateFlags(registerVal uint8, leftOrRight string) {
	c.Registers.SetFlag(registers.ZERO_FLAG, false)
	c.Registers.SetFlag(registers.SUBTRACTION_FLAG, false)
	c.Registers.SetFlag(registers.HALF_CARRY_FLAG, false)

	switch leftOrRight {
	case "L":
		carryBit := registerVal >> 7
		if carryBit == 0 {
			c.Registers.SetFlag(registers.CARRY_FLAG, false)
		} else if carryBit == 1 {
			c.Registers.SetFlag(registers.CARRY_FLAG, true)
		}
	case "R":
		carryBit := registerVal & 0x1
		if carryBit == 0 {
			c.Registers.SetFlag(registers.CARRY_FLAG, false)
		} else if carryBit == 1 {
			c.Registers.SetFlag(registers.CARRY_FLAG, true)
		}
	}
}

func HalfCarryAdd16(a uint16, b uint16) bool {
	// a 0000111000000000
	// b 0000001000000000
	// a & 0x0FFF = 0000111000000000
	// b & 0x0FFF = 0000001000000000
	//            +=0001000000000000
	// 0001000000000000 & 0x1000 = 0001000000000000
	// overflow from bit 11
	if ((a&0x0FFF)+(b&0x0FFF))&0x1000 == 0x1000 {
		return true
	} else {
		return false
	}
}

func FullCarryAdd16(a uint16, b uint16) bool {
	if a+b < a || a+b < b {
		return true
	} else {
		return false
	}
}

func (c *CPU) SetAddFlags16(a uint16, b uint16) {
	c.Registers.SetFlag(registers.SUBTRACTION_FLAG, false)

	if HalfCarryAdd16(a, b) {
		c.Registers.SetFlag(registers.HALF_CARRY_FLAG, true)
	}

	if FullCarryAdd16(a, b) {
		c.Registers.SetFlag(registers.CARRY_FLAG, true)
	}
}

var Instructions = [0x0100]Instruction{
	0x00: {
		Mnemonic: "NOP",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) {
		},
	},
	0x01: {
		Mnemonic: "LD_BC_N16",
		Size:     3,
		Ticks:    []uint8{12},
		AddrMode: R_A16,
		Operation: func(c *CPU) {
			c.Registers.SetBC(c.Fetched)
		},
	},
	0x02: {
		Mnemonic: "LD_[BC]_A",
		Size:     1,
		Ticks:    []uint8{8},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			c.Write(c.Registers.GetBC(), c.Registers.A)
			c.cpuCycles(1)
		},
	},
	0x03: {
		Mnemonic: "INC_BC",
		Size:     1,
		Ticks:    []uint8{8},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			c.Registers.SetBC(c.Registers.GetBC() + 1)
			c.cpuCycles(1)
		},
	},
	0x04: {
		Mnemonic: "INC_B",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			c.SetIncFlags(c.Registers.B)
			c.Registers.B++
		},
	},
	0x05: {
		Mnemonic: "DEC_B",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			c.SetDecFlags(c.Registers.B)
			c.Registers.B--
		},
	},
	0x06: {
		Mnemonic: "LD_B_N8",
		Size:     2,
		Ticks:    []uint8{8},
		AddrMode: R_N8,
		Operation: func(c *CPU) {
			c.Registers.B = uint8(c.Fetched)
		},
	},
	0x07: {
		Mnemonic: "RLCA",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			a := c.Registers.A
			c.SetRotateFlags(a, "L")
			// a = 11101000
			// a << 1 = 11010000
			// a >> 7 = 00000001
			//          11010001
			c.Registers.A = (a << 1) | (a >> 7)
		},
	},
	0x08: {
		Mnemonic: "LD_[A16]_SP",
		Size:     3,
		Ticks:    []uint8{20},
		AddrMode: R_A16,
		Operation: func(c *CPU) {
			lo := uint8(c.Registers.SP & 0x00FF)
			hi := uint8(c.Registers.SP >> 8)
			c.Write(c.Fetched, lo)
			c.cpuCycles(1)
			c.Write(c.Fetched+1, hi)
		},
	},
	0x09: {
		Mnemonic: "ADD_HL_BC",
		Size:     1,
		Ticks:    []uint8{8},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			c.SetAddFlags16(c.Registers.GetHL(), c.Registers.GetBC())
			c.Registers.SetHL(c.Registers.GetHL() + c.Registers.GetBC())
			c.cpuCycles(1)
		},
	},
	0x0A: {
		Mnemonic: "LD_A_[BC]",
		Size:     1,
		Ticks:    []uint8{8},
		AddrMode: R_A16,
		Operation: func(c *CPU) {
			c.Registers.A = c.Read(c.Fetched)
		},
	},
	0x0B: {
		Mnemonic: "DEC_BC",
		Size:     1,
		Ticks:    []uint8{8},
		AddrMode: R_A16,
		Operation: func(c *CPU) {
			c.Registers.SetBC(c.Registers.GetBC() - 1)
			c.cpuCycles(1)
		},
	},
	0x0C: {
		Mnemonic: "INC_C",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			c.SetIncFlags(c.Registers.C)
			c.Registers.C++
		},
	},
	0x0D: {
		Mnemonic: "DEC_C",
		Size:     1,
		AddrMode: NONE,
		Ticks:    []uint8{4},
		Operation: func(c *CPU) {
			c.SetDecFlags(c.Registers.C)
			c.Registers.C--
		},
	},
	0x0E: {
		Mnemonic: "LD_C_N8",
		Size:     2,
		AddrMode: R_N8,
		Operation: func(c *CPU) {
			c.Registers.C = uint8(c.Fetched)
		},
	},
	0x0F: {
		Mnemonic: "RRCA",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			a := c.Registers.A
			c.SetRotateFlags(a, "R")
			// a = 11101001
			// a >> 1 = 01110100
			// a << 7 = 10000000
			//          11110100
			c.Registers.A = (a >> 1) | (a << 7)
		},
	},
	0x10: {},
	0x11: {},
	0x12: {},
	0x13: {},
	0x14: {},
	0x15: {},
	0x16: {},
	0x17: {},
	0x18: {},
	0x19: {},
	0x1A: {},
	0x1B: {},
	0x1C: {},
	0x1D: {},
	0x1E: {},
	0x1F: {},
	0x20: {
		Mnemonic: "JR_NZ_E8",
		Size:     2,
		AddrMode: E8,
		Operation: func(c *CPU) {
			if !c.Registers.GetFlag(registers.ZERO_FLAG) {
				var pc uint16 = c.Registers.PC + uint16(c.RelAddr)
				c.Registers.PC = pc
				c.cpuCycles(1)
			}
		},
	},
	0x21: {
		Mnemonic: "LD_HL_N16",
		Size:     3,
		AddrMode: R_A16,
		Operation: func(c *CPU) {
			c.Registers.SetHL(c.Fetched)
			c.cpuCycles(1)
		},
	},
	0x22: {},
	0x23: {},
	0x24: {},
	0x25: {},
	0x26: {},
	0x27: {},
	0x28: {},
	0x29: {},
	0x2A: {},
	0x2B: {},
	0x2C: {},
	0x2D: {},
	0x2E: {},
	0x2F: {},
	0x30: {},
	0x31: {
		Mnemonic: "LD_SP_N16",
		Size:     3,
		AddrMode: R_A16,
		Operation: func(c *CPU) {
			c.Registers.SP = c.Fetched
		},
	},
	0x32: {
		Mnemonic: "LD_[HLD]_A",
		Size:     1,
		AddrMode: NONE,
		Operation: func(c *CPU) {
			c.Write(c.Registers.GetHL(), c.Registers.A)
			prev := c.Registers.GetHL()
			c.Registers.SetHL(prev - 1)
		},
	},
	0x33: {},
	0x34: {},
	0x35: {},
	0x36: {},
	0x37: {},
	0x38: {},
	0x39: {},
	0x3A: {},
	0x3B: {},
	0x3C: {},
	0x3D: {},
	0x3E: {
		Mnemonic: "LD_A_N8",
		Size:     2,
		AddrMode: R_N8,
		Operation: func(c *CPU) {
			c.Registers.A = uint8(c.Fetched)
		},
	},
	0x3F: {},
	0x40: {},
	0x41: {},
	0x42: {},
	0x43: {},
	0x44: {},
	0x45: {},
	0x46: {},
	0x47: {},
	0x48: {},
	0x49: {},
	0x4A: {},
	0x4B: {},
	0x4C: {},
	0x4D: {},
	0x4E: {},
	0x4F: {},
	0x50: {},
	0x51: {},
	0x52: {},
	0x53: {},
	0x54: {},
	0x55: {},
	0x56: {},
	0x57: {},
	0x58: {},
	0x59: {},
	0x5A: {},
	0x5B: {},
	0x5C: {},
	0x5D: {},
	0x5E: {},
	0x5F: {},
	0x60: {},
	0x61: {},
	0x62: {},
	0x63: {},
	0x64: {},
	0x65: {},
	0x66: {},
	0x67: {},
	0x68: {},
	0x69: {},
	0x6A: {},
	0x6B: {},
	0x6C: {},
	0x6D: {},
	0x6E: {},
	0x6F: {},
	0x70: {},
	0x71: {},
	0x72: {},
	0x73: {},
	0x74: {},
	0x75: {},
	0x76: {},
	0x77: {},
	0x78: {},
	0x79: {},
	0x7A: {},
	0x7B: {},
	0x7C: {},
	0x7D: {},
	0x7E: {},
	0x7F: {},
	0x80: {},
	0x81: {},
	0x82: {},
	0x83: {},
	0x84: {},
	0x85: {},
	0x86: {},
	0x87: {},
	0x88: {},
	0x89: {},
	0x8A: {},
	0x8B: {},
	0x8C: {},
	0x8D: {},
	0x8E: {},
	0x8F: {},
	0x90: {},
	0x91: {},
	0x92: {},
	0x93: {},
	0x94: {},
	0x95: {},
	0x96: {},
	0x97: {},
	0x98: {},
	0x99: {},
	0x9A: {},
	0x9B: {},
	0x9C: {},
	0x9D: {},
	0x9E: {},
	0x9F: {},
	0xA0: {},
	0xA1: {},
	0xA2: {},
	0xA3: {},
	0xA4: {},
	0xA5: {},
	0xA6: {},
	0xA7: {},
	0xA8: {},
	0xA9: {},
	0xAA: {},
	0xAB: {},
	0xAC: {},
	0xAD: {},
	0xAE: {},
	0xAF: {
		Mnemonic: "XOR_A_A",
		Size:     1,
		AddrMode: NONE,
		Operation: func(c *CPU) {
			c.Registers.A = c.Registers.A ^ c.Registers.A
			c.Registers.SetFlag(registers.ZERO_FLAG, true)
		},
	},
	0xB0: {},
	0xB1: {},
	0xB2: {},
	0xB3: {},
	0xB4: {},
	0xB5: {},
	0xB6: {},
	0xB7: {},
	0xB8: {},
	0xB9: {},
	0xBA: {},
	0xBB: {},
	0xBC: {},
	0xBD: {},
	0xBE: {},
	0xBF: {},
	0xC0: {},
	0xC1: {},
	0xC2: {},
	0xC3: {
		Mnemonic: "JP_A16",
		Size:     3,
		AddrMode: R_A16,
		Operation: func(c *CPU) {
			c.Registers.PC = c.Fetched
			c.cpuCycles(1)
		},
	},
	0xC4: {},
	0xC5: {},
	0xC6: {},
	0xC7: {},
	0xC8: {},
	0xC9: {},
	0xCA: {},
	0xCB: {},
	0xCC: {},
	0xCD: {
		Mnemonic: "CALL_A16",
		Size:     3,
		AddrMode: A16_R,
		Operation: func(c *CPU) {
			c.StackPush16(c.DestAddr)
		},
	},
	0xCE: {},
	0xCF: {},
	0xD0: {},
	0xD1: {},
	0xD2: {},
	0xD3: {},
	0xD4: {},
	0xD5: {},
	0xD6: {},
	0xD7: {},
	0xD8: {},
	0xD9: {},
	0xDA: {},
	0xDB: {},
	0xDC: {},
	0xDD: {},
	0xDE: {},
	0xDF: {},
	0xE0: {
		Mnemonic: "LDH_[A8]_A",
		Size:     2,
		AddrMode: A8_A,
		Operation: func(c *CPU) {
			c.Write(c.Fetched, c.Registers.A)
			c.cpuCycles(1)
		},
	},
	0xE1: {},
	0xE2: {},
	0xE3: {},
	0xE4: {},
	0xE5: {},
	0xE6: {},
	0xE7: {},
	0xE8: {},
	0xE9: {},
	0xEA: {
		Mnemonic: "LD_[A16]_A",
		Size:     3,
		AddrMode: A16_R,
		Operation: func(c *CPU) {
			c.Write(c.Fetched, c.Registers.A)
			c.cpuCycles(1)
		},
	},
	0xEB: {},
	0xEC: {},
	0xED: {},
	0xEE: {},
	0xEF: {},
	0xF0: {
		Mnemonic: "LDH_A_[A8]",
		Size:     2,
		AddrMode: A_A8,
		Operation: func(c *CPU) {
			c.Registers.A = c.Read(c.Fetched)
			c.cpuCycles(1)
		},
	},
	0xF1: {},
	0xF2: {},
	0xF3: {
		Mnemonic: "DI",
		Size:     1,
		AddrMode: NONE,
		Operation: func(c *CPU) {
			c.Registers.IME = false
		},
	},
	0xF4: {},
	0xF5: {},
	0xF6: {},
	0xF7: {},
	0xF8: {},
	0xF9: {
		Mnemonic: "LD_SP_HL",
		Size:     1,
		AddrMode: NONE,
		Operation: func(c *CPU) {
			c.Registers.SP = c.Registers.GetHL()
		},
	},
	0xFA: {},
	0xFB: {},
	0xFC: DASH,
	0xFD: {},
	0xFE: {},
	0xFF: {},
}

var DASH = Instruction{
	Mnemonic: "-",
	Size:     1,
	AddrMode: NONE,
	Operation: func(c *CPU) {
	},
}
