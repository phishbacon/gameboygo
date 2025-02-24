package cpu

import (
	"fmt"
	"goboy/bus"
	"goboy/cpu/registers"
	"os"
)

type CPU struct {
	bus       *bus.Bus
	registers *registers.Registers
	curInst   *Instruction

	fetched          uint16
	absAddr          uint16
	relAddr          int8
	destReg          string
	cycles           uint8
	intrMasterEnable bool
	halted           bool
}

func NewCPU(bus *bus.Bus) *CPU {
	registers := new(registers.Registers)
	return &CPU{
		bus:       bus,
		registers: registers,
	}
}

func (c *CPU) Init() {
	c.registers.PC = 0x0100
}

func (c *CPU) Read(address uint16) uint8 {
	return c.bus.Read(address)
}

func (c *CPU) Write(address uint16, value uint8) {
	c.bus.Write(address, value)
}

func (c *CPU) StackPush(value uint8) {
	c.registers.SP--
	c.Write(c.registers.SP, value)
}

func (c *CPU) StackPush16(value uint16) {
	// push lo
	c.StackPush(uint8(value<<8) & 0x00FF)
	// push hi
	c.StackPush(uint8(value) & 0x00FF)
}

func (c *CPU) execute() {
	pc := c.registers.PC
	opcode := c.Read(pc)
	if opcode >= uint8(len(Instructions)) {
		fmt.Printf("opcode: 0x%04x undefined\n", opcode)
		os.Exit(-1)
	} else if Instructions[opcode].Operation == nil {
		fmt.Printf("opcode: 0x%04x not implemented\n", opcode)
		fmt.Printf("0x%04x 0x02%d 0x02%d\n", opcode, c.Read(c.registers.PC+1), c.Read(c.registers.PC+2))
		os.Exit(-1)
	}

	c.process(opcode)
}

func (c *CPU) process(opcode uint8) {
	c.curInst = &Instructions[opcode]
	fmt.Printf("%10s(0x%02x) 0x%02x 0x%02x ",
		c.curInst.Mnemonic,
		opcode,
		c.Read(c.registers.PC+1),
		c.Read(c.registers.PC+2))
	c.registers.PC++
	c.curInst.AddrMode(c)
	c.curInst.Operation(c)
	fmt.Printf("AF: 0b%016b BC: 0x%04x DE: 0x%04x HL: 0x%04x PC: 0x%04x SP: 0x%04x\n",
		c.registers.GetAF(),
		c.registers.GetBC(),
		c.registers.GetDE(),
		c.registers.GetHL(),
		c.registers.PC,
		c.registers.SP)
}

func (c *CPU) Step() bool {
	if !c.halted {
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

// no operation
func NONE(c *CPU) {
	return
}

// 16 bit address
func A16(c *CPU) {
	// grab low and hi byte from adddress pc and pc +1
	lo := c.Read(c.registers.PC)
	hi := c.Read(c.registers.PC + 1)
	c.registers.PC += 2
	addr := (uint16(hi) << 8) | uint16(lo)
	c.absAddr = addr
}

func E8(c *CPU) {
	c.relAddr = int8(c.Read(c.registers.PC))
}

// 16 bit immediate data
func N16(c *CPU) {
	// grab low and hi byte from pc and pc +1
	lo := c.Read(c.registers.PC)
	hi := c.Read(c.registers.PC + 1)
	c.registers.PC += 2
	data := (uint16(hi) << 8) | uint16(lo)
	c.fetched = data
}

// 8 bit immediate data
func N8(c *CPU) {
	// grab low and hi byte from pc and pc +1
	lo := c.Read(c.registers.PC)
	c.registers.PC += 1
	c.fetched = uint16(lo)
}

func A8(c *CPU) {
	lo := c.Read(c.registers.PC)
	c.registers.PC += 1
	c.fetched = uint16(lo)
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
	c.registers.SetFlag(registers.SUBTRACTION_FLAG, true)
	if registerVal-1 == 0 {
		c.registers.SetFlag(registers.ZERO_FLAG, true)
	}
	if HalfCarrySub(registerVal, registerVal-1) {
		c.registers.SetFlag(registers.HALF_CARRY_FLAG, true)
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
			c.SetDecFlags(c.registers.B)
			c.registers.B--
			return 4
		},
	},
	0x06: {
		Mnemonic: "LD_B_N8",
		Size:     2,
		Ticks:    []uint8{8},
		AddrMode: N8,
		Operation: func(c *CPU) int {
			c.registers.B = uint8(c.fetched)
			return 8
		},
	},
	0x0E: {
		Mnemonic: "LD_C_N8",
		Size:     2,
		Ticks:    []uint8{8},
		AddrMode: N8,
		Operation: func(c *CPU) int {
			c.registers.C = uint8(c.fetched)
			return 8
		},
	},
	0x21: {
		Mnemonic: "LD_HL_N16",
		Size:     3,
		AddrMode: N16,
		Operation: func(c *CPU) int {
			c.registers.SetHL(c.fetched)
			return 12
		},
	},
	0x32: {
		Mnemonic: "LD_[HLD]_A",
		Size:     1,
		AddrMode: NONE,
		Operation: func(c *CPU) int {
			c.Write(c.registers.GetHL(), c.registers.A)
			prev := c.registers.GetHL()
			c.registers.SetHL(prev - 1)
			return 8
		},
	},
	0xC3: {
		Mnemonic: "JP_A16",
		Size:     3,
		AddrMode: A16,
		Operation: func(c *CPU) int {
			c.registers.PC = c.absAddr
			return 16
		},
	},
	0x31: {
		Mnemonic: "LD_SP_N16",
		Size:     3,
		AddrMode: N16,
		Operation: func(c *CPU) int {
			c.registers.SP = c.fetched
			return 12
		},
	},
	0xAF: {
		Mnemonic: "XOR_A_A",
		Size:     1,
		AddrMode: NONE,
		Operation: func(c *CPU) int {
			c.registers.A = c.registers.A ^ c.registers.A
			c.registers.SetFlag(registers.ZERO_FLAG, true)
			return 4
		},
	},
	0xF3: {
		Mnemonic: "DI",
		Size:     1,
		AddrMode: NONE,
		Operation: func(c *CPU) int {
			c.intrMasterEnable = false
			return 4
		},
	},
	0x20: {
		Mnemonic: "JR_NZ_E8",
		Size:     2,
		AddrMode: E8,
		Operation: func(c *CPU) int {
			if !c.registers.GetFlag(registers.ZERO_FLAG) {
				var pc uint16 = c.registers.PC + uint16(c.relAddr)
				c.registers.PC = pc
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
			c.SetDecFlags(c.registers.C)
			c.registers.C--
			return 4
		},
	},
	0xF9: {
		Mnemonic: "LD_SP_HL",
		Size:     1,
		AddrMode: NONE,
		Operation: func(c *CPU) int {
			c.registers.SP = c.registers.GetHL()
			return 8
		},
	},
	0x3E: {
		Mnemonic: "LD_A_N8",
		Size:     2,
		AddrMode: N8,
		Operation: func(c *CPU) int {
			c.registers.A = uint8(c.fetched)
			return 8
		},
	},
	0xEA: {
		Mnemonic: "LD_[A16]_A",
		Size:     3,
		AddrMode: A16,
		Operation: func(c *CPU) int {
			c.Write(c.absAddr, c.registers.A)
			return 16
		},
	},
	0xE0: {
		Mnemonic: "LDH_[A8]_A",
		Size:     2,
		AddrMode: A8,
		Operation: func(c *CPU) int {
			c.Write(0xFF00+c.fetched, c.registers.A)
			return 12
		},
	},
	0xF0: {
		Mnemonic: "LDH_A_[A8]",
		Size:     2,
		AddrMode: NONE,
		Operation: func(c *CPU) int {
			fmt.Printf("%x", 0xff00+c.fetched)
			c.registers.A = c.Read(0xFF00 + c.fetched)
			return 12
		},
	},
	0xCD: {
		Mnemonic: "CALL_A16",
		Size:     3,
		AddrMode: A16,
		Operation: func(c *CPU) int {
			c.StackPush16(c.absAddr)
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
