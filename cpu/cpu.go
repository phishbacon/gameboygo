package cpu

import (
	"fmt"
	"goboy/cpu/registers"
	"os"
)

type CPU struct {
	Registers *registers.Registers
	CurInst   *Instruction

	Fetched          uint16
	DestAddr         uint16
	RelAddr          int8
	Ticks            uint64
	Halted           bool
  Read            func(uint16) uint8
  Write           func(uint16, uint8)
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
	// push lo
	c.StackPush(uint8(value<<8) & 0x00FF)
	// push hi
	c.StackPush(uint8(value) & 0x00FF)
}

func (c *CPU) execute() {
	pc := c.Registers.PC
	opcode := c.Read(pc)
	c.cpuCycles(1)
	if opcode >= uint8(len(Instructions)) {
		fmt.Printf("opcode: %04x undefined\n", opcode)
		os.Exit(-1)
	} else if Instructions[opcode].Operation == nil {
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

type Operation func(c *CPU) int
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

func A8_R(c *CPU) {
	lo := uint16(c.Read(c.Registers.PC)) + 0xFF00
	c.cpuCycles(1)
	c.Registers.PC += 1
	c.Fetched = lo
}

func R_A8(c *CPU) {
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

func (c *CPU) SetDecFlags(registerVal uint8) {
	c.Registers.SetFlag(registers.SUBTRACTION_FLAG, true)
	if registerVal-1 == 0 {
		c.Registers.SetFlag(registers.ZERO_FLAG, true)
	}
	if HalfCarrySub(registerVal, registerVal-1) {
		c.Registers.SetFlag(registers.HALF_CARRY_FLAG, true)
	}
}

var Instructions = [0x00FF]Instruction{
	0x00: {
		Mnemonic: "NOP",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) int {
			return 4
		},
	},
	0x05: {
		Mnemonic: "DEC_B",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) int {
			c.SetDecFlags(c.Registers.B)
			c.Registers.B--
			return 4
		},
	},
	0x06: {
		Mnemonic: "LD_B_N8",
		Size:     2,
		Ticks:    []uint8{8},
		AddrMode: R_N8,
		Operation: func(c *CPU) int {
			c.Registers.B = uint8(c.Fetched)
			return 8
		},
	},
	0x0E: {
		Mnemonic: "LD_C_N8",
		Size:     2,
		Ticks:    []uint8{8},
		AddrMode: R_N8,
		Operation: func(c *CPU) int {
			c.Registers.C = uint8(c.Fetched)
			return 8
		},
	},
	0x21: {
		Mnemonic: "LD_HL_N16",
		Size:     3,
		AddrMode: R_A16,
		Operation: func(c *CPU) int {
			c.Registers.SetHL(c.Fetched)
			c.cpuCycles(1)
			return 12
		},
	},
	0x32: {
		Mnemonic: "LD_[HLD]_A",
		Size:     1,
		AddrMode: NONE,
		Operation: func(c *CPU) int {
			c.Write(c.Registers.GetHL(), c.Registers.A)
			prev := c.Registers.GetHL()
			c.Registers.SetHL(prev - 1)
			return 8
		},
	},
	0xC3: {
		Mnemonic: "JP_A16",
		Size:     3,
		AddrMode: R_A16,
		Operation: func(c *CPU) int {
			c.Registers.PC = c.Fetched
			c.cpuCycles(1)
			return 16
		},
	},
	0x31: {
		Mnemonic: "LD_SP_N16",
		Size:     3,
		AddrMode: R_A16,
		Operation: func(c *CPU) int {
			c.Registers.SP = c.Fetched
			return 12
		},
	},
	0xAF: {
		Mnemonic: "XOR_A_A",
		Size:     1,
		AddrMode: NONE,
		Operation: func(c *CPU) int {
			c.Registers.A = c.Registers.A ^ c.Registers.A
			c.Registers.SetFlag(registers.ZERO_FLAG, true)
			return 4
		},
	},
	0xF3: {
		Mnemonic: "DI",
		Size:     1,
		AddrMode: NONE,
		Operation: func(c *CPU) int {
			c.Registers.IME = false
			return 4
		},
	},
	0x20: {
		Mnemonic: "JR_NZ_E8",
		Size:     2,
		AddrMode: E8,
		Operation: func(c *CPU) int {
			if !c.Registers.GetFlag(registers.ZERO_FLAG) {
				var pc uint16 = c.Registers.PC + uint16(c.RelAddr)
				c.Registers.PC = pc
				return 12
			}
			return 8
		},
	},
	0xFC: DASH,
	0x0D: {
		Mnemonic: "DEC_C",
		Size:     1,
		AddrMode: NONE,
		Operation: func(c *CPU) int {
			c.SetDecFlags(c.Registers.C)
			c.Registers.C--
			return 4
		},
	},
	0xF9: {
		Mnemonic: "LD_SP_HL",
		Size:     1,
		AddrMode: NONE,
		Operation: func(c *CPU) int {
			c.Registers.SP = c.Registers.GetHL()
			return 8
		},
	},
	0x3E: {
		Mnemonic: "LD_A_N8",
		Size:     2,
		AddrMode: R_N8,
		Operation: func(c *CPU) int {
			c.Registers.A = uint8(c.Fetched)
			return 8
		},
	},
	0xEA: {
		Mnemonic: "LD_[A16]_A",
		Size:     3,
		AddrMode: A16_R,
		Operation: func(c *CPU) int {
			c.Write(c.Fetched, c.Registers.A)
			c.cpuCycles(1)
			return 16
		},
	},
	0xE0: {
		Mnemonic: "LDH_[A8]_A",
		Size:     2,
		AddrMode: A8_R,
		Operation: func(c *CPU) int {
			c.Write(c.Fetched, c.Registers.A)
			c.cpuCycles(1)
			return 12
		},
	},
	0xF0: {
		Mnemonic: "LDH_A_[A8]",
		Size:     2,
		AddrMode: R_A8,
		Operation: func(c *CPU) int {
			c.Registers.A = c.Read(c.Fetched)
      c.cpuCycles(1)
			return 12
		},
	},
	0xCD: {
		Mnemonic: "CALL_A16",
		Size:     3,
		AddrMode: A16_R,
		Operation: func(c *CPU) int {
			c.StackPush16(c.DestAddr)
			return 24
		},
	},
}

var DASH = Instruction{
	Mnemonic: "-",
	Size:     1,
	AddrMode: NONE,
	Operation: func(c *CPU) int {
		return 4
	},
}
