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
	c.cpuCycles(1)
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
	return ((a&0x0f)-(b&0x0f))&0x10 == 0x10
}

func FullCarrySub(a uint8, b uint8) bool {
	return a-b > a || a-b > b
}

func HalfCarrySbc(a uint8, b uint8, c uint8) bool {
	return ((a&0x0f)-(b&0x0f)-(c&0x0f))&0x10 == 0x10
}

func FullCarrySbc(a uint8, b uint8, c uint8) bool {
	return b + c > a
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
	return ((a&0x0f)+(b&0x0f))&0x10 == 0x10
}

func FullCarryAdd(a uint8, b uint8) bool {
	return a+b < a || a+b < b
}

func HalfCarryAdc(a uint8, b uint8, c uint8) bool {
	return ((a&0x0f)+(b&0x0f)+(c&0x0f))&0x10 == 0x10
}

func FullCarryAdc(a uint8, b uint8, c uint8) bool {
	return a+b+c < a || a+b+c < b || a+b+c < c
}

func (c *CPU) SetDecRegFlags(registerRef *uint8) {
	c.Registers.SetFlag(registers.SUBTRACTION_FLAG, true)
	c.Registers.SetFlag(registers.ZERO_FLAG, *registerRef-1 == 0)
	c.Registers.SetFlag(registers.HALF_CARRY_FLAG, *registerRef&0x000F == 0x0000)
	(*registerRef)--
}

func (c *CPU) SetIncRegFlags(registerRef *uint8) {
	c.Registers.SetFlag(registers.SUBTRACTION_FLAG, false)
	c.Registers.SetFlag(registers.ZERO_FLAG, *registerRef+1 == 0)
	c.Registers.SetFlag(registers.HALF_CARRY_FLAG, *registerRef&0x000F == 0x000F)
	(*registerRef)++
}

func (c *CPU) SetDecFlags(value uint8) {
	c.Registers.SetFlag(registers.SUBTRACTION_FLAG, true)
	c.Registers.SetFlag(registers.ZERO_FLAG, value-1 == 0)
	c.Registers.SetFlag(registers.HALF_CARRY_FLAG, value&0x000F == 0x0000)
}

func (c *CPU) SetIncFlags(value uint8) {
	c.Registers.SetFlag(registers.SUBTRACTION_FLAG, false)
	c.Registers.SetFlag(registers.ZERO_FLAG, value+1 == 0)
	c.Registers.SetFlag(registers.HALF_CARRY_FLAG, value&0x000F == 0x000F)
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
		carryBit := registerVal & 0x0001
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
	return ((a&0x0FFF)+(b&0x0FFF))&0x1000 == 0x1000
}

func FullCarryAdd16(a uint16, b uint16) bool {
	return a+b < a || a+b < b
}

func (c *CPU) SetAddFlags(a, b uint8) {
	c.Registers.SetFlag(registers.ZERO_FLAG, a+b == 0)
	c.Registers.SetFlag(registers.SUBTRACTION_FLAG, false)
	c.Registers.SetFlag(registers.HALF_CARRY_FLAG, HalfCarryAdd(a, b))
	c.Registers.SetFlag(registers.CARRY_FLAG, FullCarryAdd(a, b))
}

func (c *CPU) SetSubFlags(a, b uint8) {
	c.Registers.SetFlag(registers.ZERO_FLAG, a-b == 0)
	c.Registers.SetFlag(registers.SUBTRACTION_FLAG, true)
	c.Registers.SetFlag(registers.HALF_CARRY_FLAG, HalfCarrySub(a, b))
	c.Registers.SetFlag(registers.CARRY_FLAG, FullCarrySub(a, b))
}

func (c *CPU) SetAdcFlags(a, b uint8) uint8 {
	var carryFlag uint8
	if c.Registers.GetFlag(registers.CARRY_FLAG) {
		carryFlag = 1
	}
	c.Registers.SetFlag(registers.ZERO_FLAG, a+b+carryFlag == 0)
	c.Registers.SetFlag(registers.SUBTRACTION_FLAG, false)
	c.Registers.SetFlag(registers.HALF_CARRY_FLAG, HalfCarryAdc(a, b, carryFlag))
	c.Registers.SetFlag(registers.CARRY_FLAG, FullCarryAdc(a, b, carryFlag))

	return carryFlag
}

func (c *CPU) SetSbcFlags(a, b uint8) uint8 {
	var carryFlag uint8
	if c.Registers.GetFlag(registers.CARRY_FLAG) {
		carryFlag = 1
	}
	c.Registers.SetFlag(registers.ZERO_FLAG, a-b-carryFlag == 0)
	c.Registers.SetFlag(registers.SUBTRACTION_FLAG, true)
	c.Registers.SetFlag(registers.HALF_CARRY_FLAG, HalfCarrySbc(a, b, carryFlag))
	c.Registers.SetFlag(registers.CARRY_FLAG, FullCarrySbc(a, b, carryFlag))

	return carryFlag
}

func (c *CPU) SetAddFlags16(a uint16, b uint16) {
	c.Registers.SetFlag(registers.SUBTRACTION_FLAG, false)
	c.Registers.SetFlag(registers.HALF_CARRY_FLAG, HalfCarryAdd16(a, b))
	c.Registers.SetFlag(registers.CARRY_FLAG, FullCarryAdd16(a, b))
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
			c.SetIncRegFlags(&c.Registers.B)
		},
	},
	0x05: {
		Mnemonic: "DEC_B",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			c.SetDecRegFlags(&c.Registers.B)
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
		AddrMode: NONE,
		Operation: func(c *CPU) {
			c.Registers.A = c.Read(c.Registers.GetBC())
			c.cpuCycles(1)
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
			c.SetIncRegFlags(&c.Registers.C)
		},
	},
	0x0D: {
		Mnemonic: "DEC_C",
		Size:     1,
		AddrMode: NONE,
		Ticks:    []uint8{4},
		Operation: func(c *CPU) {
			c.SetDecRegFlags(&c.Registers.C)
		},
	},
	0x0E: {
		Mnemonic: "LD_C_N8",
		Size:     2,
		Ticks:    []uint8{8},
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
	0x10: {
		Mnemonic: "STOP_N8",
		Size:     2,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			fmt.Fprint(os.Stderr, "STOP!\n")
		},
	},
	0x11: {
		Mnemonic: "LD_DE_N16",
		Size:     3,
		Ticks:    []uint8{12},
		AddrMode: R_A16,
		Operation: func(c *CPU) {
			c.Registers.SetDE(c.Fetched)
		},
	},
	0x12: {
		Mnemonic: "LD_[DE]_A",
		Size:     1,
		Ticks:    []uint8{8},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			c.Write(c.Registers.GetDE(), c.Registers.A)
			c.cpuCycles(1)
		},
	},
	0x13: {
		Mnemonic: "INC_DE",
		Size:     1,
		Ticks:    []uint8{8},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			c.Registers.SetDE(c.Registers.GetDE() + 1)
			c.cpuCycles(1)
		},
	},
	0x14: {
		Mnemonic: "INC_D",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			c.SetIncRegFlags(&c.Registers.D)
		},
	},
	0x15: {
		Mnemonic: "DEC_D",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			c.SetDecRegFlags(&c.Registers.D)
		},
	},
	0x16: {
		Mnemonic: "LD_D_N8",
		Size:     2,
		Ticks:    []uint8{8},
		AddrMode: R_N8,
		Operation: func(c *CPU) {
			c.Registers.D = uint8(c.Fetched)
		},
	},
	0x17: {
		Mnemonic: "RLA",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			a := c.Registers.A
			var oldCarry uint8
			if c.Registers.GetFlag(registers.CARRY_FLAG) {
				oldCarry = 1
			}
			c.SetRotateFlags(a, "L")
			// oldCarry = 1
			// a = 10010100
			// a << 1 = 00101000
			// a | 00000001 = 00101001
			c.Registers.A = (a << 1) | oldCarry
		},
	},
	0x18: {
		Mnemonic: "JR_E8",
		Size:     2,
		Ticks:    []uint8{12},
		AddrMode: E8,
		Operation: func(c *CPU) {
			c.Registers.PC += uint16(c.RelAddr)
			c.cpuCycles(1)
		},
	},
	0x19: {
		Mnemonic: "ADD_HL_DE",
		Size:     1,
		Ticks:    []uint8{8},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			c.SetAddFlags16(c.Registers.GetHL(), c.Registers.GetDE())
			c.Registers.SetHL(c.Registers.GetHL() + c.Registers.GetDE())
			c.cpuCycles(1)
		},
	},
	0x1A: {
		Mnemonic: "LD_A_[DE]",
		Size:     1,
		Ticks:    []uint8{8},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			fetched := c.Read(c.Registers.GetDE())
			c.cpuCycles(1)
			c.Registers.A = fetched
		},
	},
	0x1B: {
		Mnemonic: "DEC_DE",
		Size:     1,
		Ticks:    []uint8{8},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			c.Registers.SetDE(c.Registers.GetDE() - 1)
			c.cpuCycles(1)
		},
	},
	0x1C: {
		Mnemonic: "INC_E",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			c.SetIncRegFlags(&c.Registers.E)
		},
	},
	0x1D: {
		Mnemonic: "DEC_E",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			c.SetDecRegFlags(&c.Registers.E)
		},
	},
	0x1E: {
		Mnemonic: "LD_E_N8",
		Size:     2,
		Ticks:    []uint8{8},
		AddrMode: R_N8,
		Operation: func(c *CPU) {
			c.Registers.E = uint8(c.Fetched)
		},
	},
	0x1F: {
		Mnemonic: "RRA",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			a := c.Registers.A
			var oldCarry uint8
			if c.Registers.GetFlag(registers.CARRY_FLAG) {
				oldCarry = 1
			}
			c.SetRotateFlags(a, "R")
			// oldCarry = 1
			// a = 10010100
			// a >> 1 = 01001010
			// oldCarry << 7 = 10000000
			// a | 10000000 = 11001010
			c.Registers.A = (a >> 1) | (oldCarry << 7)
		},
	},
	0x20: {
		Mnemonic: "JR_NZ_E8",
		Size:     2,
		Ticks:    []uint8{12, 8},
		AddrMode: E8,
		Operation: func(c *CPU) {
			if !c.Registers.GetFlag(registers.ZERO_FLAG) {
				c.Registers.PC += uint16(c.RelAddr)
				c.cpuCycles(1)
			}
		},
	},
	0x21: {
		Mnemonic: "LD_HL_N16",
		Size:     3,
		Ticks:    []uint8{12},
		AddrMode: R_A16,
		Operation: func(c *CPU) {
			c.Registers.SetHL(c.Fetched)
		},
	},
	0x22: {
		Mnemonic: "LD_[HLI]_A",
		Size:     1,
		Ticks:    []uint8{8},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			hl := c.Registers.GetHL()
			c.Write(hl, c.Registers.A)
			c.Registers.SetHL(hl + 1)
			c.cpuCycles(1)
		},
	},
	0x23: {
		Mnemonic: "INC_HL",
		Size:     1,
		Ticks:    []uint8{8},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			c.Registers.SetHL(c.Registers.GetHL() + 1)
			c.cpuCycles(1)
		},
	},
	0x24: {
		Mnemonic: "INC_H",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			c.SetIncRegFlags(&c.Registers.H)
		},
	},
	0x25: {
		Mnemonic: "DEC_H",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			c.SetDecRegFlags(&c.Registers.H)
		},
	},
	0x26: {
		Mnemonic: "LD_H_N8",
		Size:     2,
		Ticks:    []uint8{8},
		AddrMode: R_N8,
		Operation: func(c *CPU) {
			c.Registers.H = uint8(c.Fetched)
		},
	},
	0x27: {},
	0x28: {
		Mnemonic: "JR_Z_E8",
		Size:     2,
		Ticks:    []uint8{12, 8},
		AddrMode: E8,
		Operation: func(c *CPU) {
			if c.Registers.GetFlag(registers.ZERO_FLAG) {
				c.Registers.PC += uint16(c.RelAddr)
				c.cpuCycles(1)
			}
		},
	},
	0x29: {
		Mnemonic: "ADD_HL_HL",
		Size:     1,
		Ticks:    []uint8{8},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			hl := c.Registers.GetHL()
			c.SetAddFlags16(hl, hl)
			c.Registers.SetHL(hl + hl)
			c.cpuCycles(1)
		},
	},
	0x2A: {
		Mnemonic: "LD_A_[HLI]",
		Size:     1,
		Ticks:    []uint8{8},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			fetched := c.Read(c.Registers.GetHL())
			c.cpuCycles(1)
			c.Registers.A = fetched
			c.Registers.SetHL(c.Registers.GetHL() + 1)
		},
	},
	0x2B: {
		Mnemonic: "DEC_HL",
		Size:     1,
		Ticks:    []uint8{8},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			c.Registers.SetDE(c.Registers.GetHL() - 1)
			c.cpuCycles(1)
		},
	},
	0x2C: {
		Mnemonic: "INC_L",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			c.SetIncRegFlags(&c.Registers.L)
		},
	},
	0x2D: {
		Mnemonic: "DEC_L",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			c.SetDecRegFlags(&c.Registers.L)
		},
	},
	0x2E: {
		Mnemonic: "LD_L_N8",
		Size:     2,
		Ticks:    []uint8{8},
		AddrMode: R_N8,
		Operation: func(c *CPU) {
			c.Registers.L = uint8(c.Fetched)
		},
	},
	0x2F: {
		Mnemonic: "CPL",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			c.Registers.A = ^c.Registers.A
			c.Registers.SetFlag(registers.SUBTRACTION_FLAG, true)
			c.Registers.SetFlag(registers.HALF_CARRY_FLAG, true)
		},
	},
	0x30: {
		Mnemonic: "JR_NC_E8",
		Size:     2,
		Ticks:    []uint8{12, 8},
		AddrMode: E8,
		Operation: func(c *CPU) {
			if !c.Registers.GetFlag(registers.CARRY_FLAG) {
				c.Registers.PC += uint16(c.RelAddr)
				c.cpuCycles(1)
			}
		},
	},
	0x31: {
		Mnemonic: "LD_SP_N16",
		Size:     3,
		Ticks:    []uint8{12},
		AddrMode: R_A16,
		Operation: func(c *CPU) {
			c.Registers.SP = c.Fetched
		},
	},
	0x32: {
		Mnemonic: "LD_[HLD]_A",
		Size:     1,
		Ticks:    []uint8{8},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			hl := c.Registers.GetHL()
			c.Write(hl, c.Registers.A)
			c.Registers.SetHL(hl - 1)
			c.cpuCycles(1)
		},
	},
	0x33: {
		Mnemonic: "INC_SP",
		Size:     1,
		Ticks:    []uint8{8},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			c.Registers.SP++
			c.cpuCycles(1)
		},
	},
	0x34: {
		Mnemonic: "INC_[HL]",
		Size:     1,
		Ticks:    []uint8{12},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			val := c.Read(c.Registers.GetHL())
			c.cpuCycles(1)
			c.SetIncFlags(val)
			c.Write(c.Registers.GetHL(), val+1)
			c.cpuCycles(1)
		},
	},
	0x35: {
		Mnemonic: "DEC_[HL]",
		Size:     1,
		Ticks:    []uint8{12},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			val := c.Read(c.Registers.GetHL())
			c.cpuCycles(1)
			c.SetDecFlags(val)
			c.Write(c.Registers.GetHL(), val-1)
			c.cpuCycles(1)
		},
	},
	0x36: {
		Mnemonic: "LD_[HL]_N8",
		Size:     2,
		Ticks:    []uint8{12},
		AddrMode: R_N8,
		Operation: func(c *CPU) {
			c.Write(c.Registers.GetHL(), uint8(c.Fetched))
			c.cpuCycles(1)
		},
	},
	0x37: {
		Mnemonic: "SCF",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			c.Registers.SetFlag(registers.SUBTRACTION_FLAG, false)
			c.Registers.SetFlag(registers.HALF_CARRY_FLAG, false)
			c.Registers.SetFlag(registers.CARRY_FLAG, true)
		},
	},
	0x38: {
		Mnemonic: "JR_C_E8",
		Size:     2,
		Ticks:    []uint8{12, 8},
		AddrMode: E8,
		Operation: func(c *CPU) {
			if c.Registers.GetFlag(registers.CARRY_FLAG) {
				c.Registers.PC += uint16(c.RelAddr)
				c.cpuCycles(1)
			}
		},
	},
	0x39: {
		Mnemonic: "ADD_HL_SP",
		Size:     1,
		Ticks:    []uint8{8},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			hl := c.Registers.GetHL()
			c.SetAddFlags16(hl, c.Registers.SP)
			c.Registers.SetHL(hl + c.Registers.SP)
			c.cpuCycles(1)
		},
	},
	0x3A: {
		Mnemonic: "LD_A_[HLD]",
		Size:     1,
		Ticks:    []uint8{8},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			fetched := c.Read(c.Registers.GetHL())
			c.cpuCycles(1)
			c.Registers.A = fetched
			c.Registers.SetHL(c.Registers.GetHL() - 1)
		},
	},
	0x3B: {
		Mnemonic: "DEC_SP",
		Size:     1,
		Ticks:    []uint8{8},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			c.Registers.SP--
			c.cpuCycles(1)
		},
	},
	0x3C: {
		Mnemonic: "INC_A",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			c.SetIncRegFlags(&c.Registers.A)
		},
	},
	0x3D: {
		Mnemonic: "DEC_A",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			c.SetDecRegFlags(&c.Registers.A)
		},
	},
	0x3E: {
		Mnemonic: "LD_A_N8",
		Size:     2,
		Ticks:    []uint8{8},
		AddrMode: R_N8,
		Operation: func(c *CPU) {
			c.Registers.A = uint8(c.Fetched)
		},
	},
	0x3F: {
		Mnemonic: "CCF",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			c.Registers.SetFlag(registers.SUBTRACTION_FLAG, false)
			c.Registers.SetFlag(registers.HALF_CARRY_FLAG, false)
			og := c.Registers.GetFlag(registers.CARRY_FLAG)
			c.Registers.SetFlag(registers.CARRY_FLAG, !og)
		},
	},
	0x40: {
		Mnemonic: "LD_B_B",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			return
		},
	},
	0x41: {
		Mnemonic: "LD_B_C",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			c.Registers.B = c.Registers.C
		},
	},
	0x42: {
		Mnemonic: "LD_B_D",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			c.Registers.B = c.Registers.D
		},
	},
	0x43: {
		Mnemonic: "LD_B_E",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			c.Registers.B = c.Registers.E
		},
	},
	0x44: {
		Mnemonic: "LD_B_H",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			c.Registers.B = c.Registers.H
		},
	},
	0x45: {
		Mnemonic: "LD_B_L",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			c.Registers.B = c.Registers.L
		},
	},
	0x46: {
		Mnemonic: "LD_B_[HL]",
		Size:     1,
		Ticks:    []uint8{8},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			val := c.Read(c.Registers.GetHL())
			c.cpuCycles(1)
			c.Registers.B = val
		},
	},
	0x47: {
		Mnemonic: "LD_B_A",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			c.Registers.B = c.Registers.A
		},
	},
	0x48: {
		Mnemonic: "LD_C_B",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			c.Registers.C = c.Registers.B
		},
	},
	0x49: {
		Mnemonic: "LD_C_C",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			return
		},
	},
	0x4A: {
		Mnemonic: "LD_C_D",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			c.Registers.C = c.Registers.D
		},
	},
	0x4B: {
		Mnemonic: "LD_C_E",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			c.Registers.C = c.Registers.E
		},
	},
	0x4C: {
		Mnemonic: "LD_C_H",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			c.Registers.C = c.Registers.H
		},
	},
	0x4D: {
		Mnemonic: "LD_C_L",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			c.Registers.C = c.Registers.L
		},
	},
	0x4E: {
		Mnemonic: "LD_C_[HL]",
		Size:     1,
		Ticks:    []uint8{8},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			val := c.Read(c.Registers.GetHL())
			c.cpuCycles(1)
			c.Registers.C = val
		},
	},
	0x4F: {
		Mnemonic: "LD_C_A",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			c.Registers.C = c.Registers.A
		},
	},
	0x50: {
		Mnemonic: "LD_D_B",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			c.Registers.D = c.Registers.B
		},
	},
	0x51: {
		Mnemonic: "LD_D_C",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			c.Registers.D = c.Registers.C
		},
	},
	0x52: {
		Mnemonic: "LD_D_D",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			return
		},
	},
	0x53: {
		Mnemonic: "LD_D_E",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			c.Registers.D = c.Registers.E
		},
	},
	0x54: {
		Mnemonic: "LD_D_H",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			c.Registers.D = c.Registers.H
		},
	},
	0x55: {
		Mnemonic: "LD_D_L",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			c.Registers.D = c.Registers.L
		},
	},
	0x56: {
		Mnemonic: "LD_D_[HL]",
		Size:     1,
		Ticks:    []uint8{8},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			val := c.Read(c.Registers.GetHL())
			c.cpuCycles(1)
			c.Registers.D = val
		},
	},
	0x57: {
		Mnemonic: "LD_D_A",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			c.Registers.D = c.Registers.A
		},
	},
	0x58: {
		Mnemonic: "LD_E_B",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			c.Registers.E = c.Registers.B
		},
	},
	0x59: {
		Mnemonic: "LD_E_C",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			c.Registers.E = c.Registers.C
		},
	},
	0x5A: {
		Mnemonic: "LD_E_D",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			c.Registers.E = c.Registers.D
		},
	},
	0x5B: {
		Mnemonic: "LD_E_E",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			return
		},
	},
	0x5C: {
		Mnemonic: "LD_E_H",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			c.Registers.E = c.Registers.H
		},
	},
	0x5D: {
		Mnemonic: "LD_E_L",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			c.Registers.E = c.Registers.L
		},
	},
	0x5E: {
		Mnemonic: "LD_E_[HL]",
		Size:     1,
		Ticks:    []uint8{8},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			val := c.Read(c.Registers.GetHL())
			c.cpuCycles(1)
			c.Registers.E = val
		},
	},
	0x5F: {
		Mnemonic: "LD_E_A",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			c.Registers.E = c.Registers.A
		},
	},
	0x60: {
		Mnemonic: "LD_H_B",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			c.Registers.H = c.Registers.B
		},
	},
	0x61: {
		Mnemonic: "LD_H_C",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			c.Registers.H = c.Registers.C
		},
	},
	0x62: {
		Mnemonic: "LD_H_D",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			c.Registers.H = c.Registers.D
		},
	},
	0x63: {
		Mnemonic: "LD_H_E",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			c.Registers.H = c.Registers.E
		},
	},
	0x64: {
		Mnemonic: "LD_H_H",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			return
		},
	},
	0x65: {
		Mnemonic: "LD_H_L",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			c.Registers.H = c.Registers.L
		},
	},
	0x66: {
		Mnemonic: "LD_H_[HL]",
		Size:     1,
		Ticks:    []uint8{8},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			val := c.Read(c.Registers.GetHL())
			c.cpuCycles(1)
			c.Registers.H = val
		},
	},
	0x67: {
		Mnemonic: "LD_H_A",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			c.Registers.H = c.Registers.A
		},
	},
	0x68: {
		Mnemonic: "LD_L_B",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			c.Registers.L = c.Registers.B
		},
	},
	0x69: {
		Mnemonic: "LD_L_C",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			c.Registers.L = c.Registers.C
		},
	},
	0x6A: {
		Mnemonic: "LD_L_D",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			c.Registers.L = c.Registers.D
		},
	},
	0x6B: {
		Mnemonic: "LD_L_E",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			c.Registers.L = c.Registers.E
		},
	},
	0x6C: {
		Mnemonic: "LD_L_H",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			c.Registers.L = c.Registers.H
		},
	},
	0x6D: {
		Mnemonic: "LD_L_L",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			return
		},
	},
	0x6E: {
		Mnemonic: "LD_L_[HL]",
		Size:     1,
		Ticks:    []uint8{8},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			val := c.Read(c.Registers.GetHL())
			c.cpuCycles(1)
			c.Registers.L = val
		},
	},
	0x6F: {
		Mnemonic: "LD_L_A",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			c.Registers.L = c.Registers.A
		},
	},
	0x70: {
		Mnemonic: "LD_[HL]_B",
		Size:     1,
		Ticks:    []uint8{8},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			c.Write(c.Registers.GetHL(), c.Registers.B)
			c.cpuCycles(1)
		},
	},
	0x71: {
		Mnemonic: "LD_[HL]_C",
		Size:     1,
		Ticks:    []uint8{8},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			c.Write(c.Registers.GetHL(), c.Registers.C)
			c.cpuCycles(1)
		},
	},
	0x72: {
		Mnemonic: "LD_[HL]_D",
		Size:     1,
		Ticks:    []uint8{8},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			c.Write(c.Registers.GetHL(), c.Registers.D)
			c.cpuCycles(1)
		},
	},
	0x73: {
		Mnemonic: "LD_[HL]_E",
		Size:     1,
		Ticks:    []uint8{8},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			c.Write(c.Registers.GetHL(), c.Registers.E)
			c.cpuCycles(1)
		},
	},
	0x74: {
		Mnemonic: "LD_[HL]_H",
		Size:     1,
		Ticks:    []uint8{8},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			c.Write(c.Registers.GetHL(), c.Registers.H)
			c.cpuCycles(1)
		},
	},
	0x75: {
		Mnemonic: "LD_[HL]_L",
		Size:     1,
		Ticks:    []uint8{8},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			c.Write(c.Registers.GetHL(), c.Registers.L)
			c.cpuCycles(1)
		},
	},
	0x76: {},
	0x77: {
		Mnemonic: "LD_[HL]_A",
		Size:     1,
		Ticks:    []uint8{8},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			c.Write(c.Registers.GetHL(), c.Registers.A)
			c.cpuCycles(1)
		},
	},
	0x78: {
		Mnemonic: "LD_A_B",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			c.Registers.A = c.Registers.B
		},
	},
	0x79: {
		Mnemonic: "LD_A_C",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			c.Registers.A = c.Registers.C
		},
	},
	0x7A: {
		Mnemonic: "LD_A_D",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			c.Registers.A = c.Registers.D
		},
	},
	0x7B: {
		Mnemonic: "LD_A_E",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			c.Registers.A = c.Registers.E
		},
	},
	0x7C: {
		Mnemonic: "LD_A_H",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			c.Registers.A = c.Registers.H
		},
	},
	0x7D: {
		Mnemonic: "LD_A_L",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			c.Registers.A = c.Registers.L
		},
	},
	0x7E: {
		Mnemonic: "LD_A_[HL]",
		Size:     1,
		Ticks:    []uint8{8},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			val := c.Read(c.Registers.GetHL())
			c.cpuCycles(1)
			c.Registers.A = val
		},
	},
	0x7F: {
		Mnemonic: "LD_A_A",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			return
		},
	},
	0x80: {
		Mnemonic: "ADD_A_B",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			c.SetAddFlags(c.Registers.A, c.Registers.B)
			c.Registers.A += c.Registers.B
		},
	},
	0x81: {
		Mnemonic: "ADD_A_C",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			c.SetAddFlags(c.Registers.A, c.Registers.C)
			c.Registers.A += c.Registers.C
		},
	},
	0x82: {
		Mnemonic: "ADD_A_D",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			c.SetAddFlags(c.Registers.A, c.Registers.D)
			c.Registers.A += c.Registers.D
		},
	},
	0x83: {
		Mnemonic: "ADD_A_E",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			c.SetAddFlags(c.Registers.A, c.Registers.E)
			c.Registers.A += c.Registers.E
		},
	},
	0x84: {
		Mnemonic: "ADD_A_H",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			c.SetAddFlags(c.Registers.A, c.Registers.H)
			c.Registers.A += c.Registers.H
		},
	},
	0x85: {
		Mnemonic: "ADD_A_L",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			c.SetAddFlags(c.Registers.A, c.Registers.L)
			c.Registers.A += c.Registers.L
		},
	},
	0x86: {
		Mnemonic: "ADD_A_[HL]",
		Size:     1,
		Ticks:    []uint8{8},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			val := c.Read(c.Registers.GetHL())
			c.SetAddFlags(c.Registers.A, val)
			c.Registers.A += val
		},
	},
	0x87: {
		Mnemonic: "ADD_A_A",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			c.SetAddFlags(c.Registers.A, c.Registers.A)
			c.Registers.A += c.Registers.A
		},
	},
	0x88: {
		Mnemonic: "ADC_A_B",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			carryFlag := c.SetAdcFlags(c.Registers.A, c.Registers.B)
			c.Registers.A += (c.Registers.B + carryFlag)
		},
	},
	0x89: {
		Mnemonic: "ADC_A_C",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			carryFlag := c.SetAdcFlags(c.Registers.A, c.Registers.C)
			c.Registers.A += (c.Registers.C + carryFlag)
		},
	},
	0x8A: {
		Mnemonic: "ADC_A_D",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			carryFlag := c.SetAdcFlags(c.Registers.A, c.Registers.D)
			c.Registers.A += (c.Registers.D + carryFlag)
		},
	},
	0x8B: {
		Mnemonic: "ADC_A_E",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			carryFlag := c.SetAdcFlags(c.Registers.A, c.Registers.E)
			c.Registers.A += (c.Registers.E + carryFlag)
		},
	},
	0x8C: {
		Mnemonic: "ADC_A_H",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			carryFlag := c.SetAdcFlags(c.Registers.A, c.Registers.H)
			c.Registers.A += (c.Registers.H + carryFlag)
		},
	},
	0x8D: {
		Mnemonic: "ADC_A_L",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			carryFlag := c.SetAdcFlags(c.Registers.A, c.Registers.L)
			c.Registers.A += (c.Registers.L + carryFlag)
		},
	},
	0x8E: {
		Mnemonic: "ADC_A_[HL]",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			val := c.Read(c.Registers.GetHL())
			c.cpuCycles(1)
			carryFlag := c.SetAdcFlags(c.Registers.A, val)
			c.Registers.A += (val + carryFlag)
		},
	},
	0x8F: {
		Mnemonic: "ADC_A_A",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			carryFlag := c.SetAdcFlags(c.Registers.A, c.Registers.A)
			c.Registers.A += (c.Registers.A + carryFlag)
		},
	},
	0x90: {
		Mnemonic: "SUB_A_B",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			c.SetSubFlags(c.Registers.A, c.Registers.B)
			c.Registers.A -= c.Registers.B
		},
	},
	0x91: {
		Mnemonic: "SUB_A_C",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			c.SetSubFlags(c.Registers.A, c.Registers.C)
			c.Registers.A -= c.Registers.C
		},
	},
	0x92: {
		Mnemonic: "SUB_A_D",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			c.SetSubFlags(c.Registers.A, c.Registers.D)
			c.Registers.A -= c.Registers.D
		},
	},
	0x93: {
		Mnemonic: "SUB_A_E",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			c.SetSubFlags(c.Registers.A, c.Registers.E)
			c.Registers.A -= c.Registers.E
		},
	},
	0x94: {
		Mnemonic: "SUB_A_H",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			c.SetSubFlags(c.Registers.A, c.Registers.H)
			c.Registers.A -= c.Registers.H
		},
	},
	0x95: {
		Mnemonic: "SUB_A_L",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			c.SetSubFlags(c.Registers.A, c.Registers.L)
			c.Registers.A -= c.Registers.L
		},
	},
	0x96: {
		Mnemonic: "SUB_A_[HL]",
		Size:     1,
		Ticks:    []uint8{8},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			val := c.Read(c.Registers.GetHL())
			c.cpuCycles(1)
			c.SetSubFlags(c.Registers.A, val)
			c.Registers.A -= val
		},
	},
	0x97: {
		Mnemonic: "SUB_A_A",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			c.Registers.SetFlag(registers.ZERO_FLAG, true)
			c.Registers.SetFlag(registers.SUBTRACTION_FLAG, true)
			c.Registers.SetFlag(registers.HALF_CARRY_FLAG, false)
			c.Registers.SetFlag(registers.CARRY_FLAG, false)
			c.Registers.A = 0
		},
	},
	0x98: {
		Mnemonic: "SBC_A_B",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			carryFlag := c.SetSbcFlags(c.Registers.A, c.Registers.B)
			c.Registers.A -= (c.Registers.B - carryFlag)
		},
	},
	0x99: {
		Mnemonic: "SBC_A_C",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			carryFlag := c.SetSbcFlags(c.Registers.A, c.Registers.C)
			c.Registers.A -= (c.Registers.C - carryFlag)
		},
	},
	0x9A: {
		Mnemonic: "SBC_A_D",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			carryFlag := c.SetSbcFlags(c.Registers.A, c.Registers.D)
			c.Registers.A -= (c.Registers.D - carryFlag)
		},
	},
	0x9B: {
		Mnemonic: "SBC_A_E",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			carryFlag := c.SetSbcFlags(c.Registers.A, c.Registers.E)
			c.Registers.A -= (c.Registers.E - carryFlag)
		},
	},
	0x9C: {
		Mnemonic: "SBC_A_H",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			carryFlag := c.SetSbcFlags(c.Registers.A, c.Registers.H)
			c.Registers.A -= (c.Registers.H - carryFlag)
		},
	},
	0x9D: {
		Mnemonic: "SBC_A_L",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			carryFlag := c.SetSbcFlags(c.Registers.A, c.Registers.L)
			c.Registers.A -= (c.Registers.L - carryFlag)
		},
	},
	0x9E: {
		Mnemonic: "SBC_A_[HL]",
		Size:     1,
		Ticks:    []uint8{8},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			val := c.Read(c.Registers.GetHL())
			carryFlag := c.SetSbcFlags(c.Registers.A, val)
			c.Registers.A -= (val - carryFlag)
		},
	},
	0x9F: {
		Mnemonic: "SBC_A_A",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			oldCarryFlag := c.Registers.GetFlag(registers.CARRY_FLAG)
			carryFlag := c.SetSbcFlags(c.Registers.A, c.Registers.A)
			c.Registers.A -= (c.Registers.A - carryFlag)
			// set carryFlag back to original value as it should not be affected by this opcode
			c.Registers.SetFlag(registers.SUBTRACTION_FLAG, oldCarryFlag)
		},
	},
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
