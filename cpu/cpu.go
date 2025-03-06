package cpu

import (
	"fmt"
	"goboy/cpu/registers"
	"goboy/io"
	"os"
)

const IF uint16 = 0xFF0F
const IE uint16 = 0xFFFF

type CPU struct {
	Registers *registers.Registers
	CurInst   *Instruction

	Fetched        uint16
	Paused         bool
	DestAddr       uint16
	RelAddr        int8
	Ticks          uint64
	Halted         bool
	EnablingIME    bool
	Read           func(uint16) uint8
	Write          func(uint16, uint8)
	CPUStateString string
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
	c.Registers.SetAF(0x01B0)
	c.Registers.SetBC(0x0000)
	c.Registers.SetDE(0xFF56)
	c.Registers.SetHL(0x000D)
	c.Registers.SetSP(0xFFFE)
	c.Registers.SetPC(0x0100)
}

func (c *CPU) StackPush(value uint8) {
	c.Registers.DecSP()
	c.Write(c.Registers.GetSP(), value)
	c.cpuCycles(1)
}

func (c *CPU) StackPush16(value uint16) {
	// push hi
	c.StackPush(uint8((value & 0xFF00) >> 8))
	// push lo
	c.StackPush(uint8(value & 0x00FF))
}

func (c *CPU) StackPop() uint8 {
	poppedValue := c.Read(c.Registers.GetSP())
	c.Registers.IncSP()
	c.cpuCycles(1)
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
	pc := c.Registers.GetPC()
	opcode := c.Read(pc)
	c.cpuCycles(1)
	if Instructions[opcode].Operation == nil {
		fmt.Printf("opcode: %04x not implemented\n", opcode)
		fmt.Printf("%02x 02%d 02%d\n", opcode, c.Read(pc+1), c.Read(pc+2))
		os.Exit(-1)
	}

	c.process(opcode)
}

func (c *CPU) process(opcode uint8) {
	c.CPUStateString = ""
	c.CurInst = &Instructions[opcode]
	pc := c.Registers.GetPC()
	pc1 := c.Read(pc + 1)
	pc2 := c.Read(pc + 2)
	c.CPUStateString += fmt.Sprintf("%-10s %02x %02x %02x\t",
		c.CurInst.Mnemonic,
		opcode,
		pc1,
		pc2)
	c.Registers.IncPC()
	c.CurInst.AddrMode(c)
	c.CurInst.Operation(c)
	var z, n, h, carry string
	if c.Registers.GetFlag(registers.ZERO_FLAG) {
		z = "Z"
	} else {
		z = "-"
	}
	if c.Registers.GetFlag(registers.SUBTRACTION_FLAG) {
		n = "N"
	} else {
		n = "-"
	}
	if c.Registers.GetFlag(registers.HALF_CARRY_FLAG) {
		h = "H"
	} else {
		h = "-"
	}
	if c.Registers.GetFlag(registers.CARRY_FLAG) {
		carry = "C"
	} else {
		carry = "-"
	}
	c.CPUStateString += fmt.Sprintf("A: 0x%02x F: %s%s%s%s BC: 0x%04x DE: 0x%04x HL: 0x%04x PC: 0x%04x SP: 0x%04x SB: 0x%04x SC: 0x%04x\n",
		c.Registers.GetReg(registers.A),
		z,
		n,
		h,
		carry,
		c.Registers.GetBC(),
		c.Registers.GetDE(),
		c.Registers.GetHL(),
		c.Registers.GetPC(),
		c.Registers.GetSP(),
		c.Read(0xFF01),
		c.Read(0xFF02))
	// c.Ticks)
	fmt.Print(c.CPUStateString)
}

func (c *CPU) Step() bool {
	if !c.Halted {
		c.execute()
	} else {
		c.cpuCycles(1)
		if c.Read(IF)&c.Read(IE) > 0 {
			c.Halted = false
		}
	}

	if c.Registers.GetIME() {
		c.HandleInterupts()
		c.EnablingIME = false
	}

	if c.EnablingIME {
		c.Registers.SetIME(true)
	}
	return true
}

func (c *CPU) HandleInterupts() {
	interuptsFlag := c.Read(IF)
	interuptsEnabled := c.Read(IE)
	if c.CheckInterupt(0x40, interuptsFlag, interuptsEnabled, io.VBLANK) {

	} else if c.CheckInterupt(0x48, interuptsFlag, interuptsEnabled, io.LCD) {

	} else if c.CheckInterupt(0x50, interuptsFlag, interuptsEnabled, io.TIMER) {

	} else if c.CheckInterupt(0x58, interuptsFlag, interuptsEnabled, io.SERIAL) {

	} else if c.CheckInterupt(0x60, interuptsFlag, interuptsEnabled, io.JOYPAD) {

	}
}

func (c *CPU) CheckInterupt(address uint16, inf uint8, ie uint8, interupt uint8) bool {
	if inf&interupt > 0 && ie&interupt > 0 {
		c.CallInterupt(address, interupt)
		c.Write(IF, inf & ^interupt)
		c.Halted = false
		c.Registers.SetIME(false)
		return true
	}

	return false
}

func (c *CPU) CallInterupt(address uint16, interupt uint8) {
	c.StackPush16(c.Registers.GetPC())
	c.Registers.SetPC(address)
}

type Operation func(c *CPU)
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

// 16 bit address
func R_A16(c *CPU) {
	// grab low and hi byte from adddress pc and pc +1
	lo := c.Read(c.Registers.GetPC())
	c.cpuCycles(1)
	hi := c.Read(c.Registers.GetPC() + 1)
	c.cpuCycles(1)
	c.Registers.IncPC()
	c.Registers.IncPC()
	c.Fetched = (uint16(hi) << 8) | uint16(lo)
}

func A16_R(c *CPU) {
	// grab low and hi byte from adddress pc and pc +1
	lo := c.Read(c.Registers.GetPC())
	c.cpuCycles(1)
	hi := c.Read(c.Registers.GetPC() + 1)
	c.cpuCycles(1)
	c.Registers.IncPC()
	c.Registers.IncPC()
	c.Fetched = (uint16(hi) << 8) | uint16(lo)
}

func E8(c *CPU) {
	// fmt.Printf("0x%04x relAddr: 0x%04x(%d)", c.Registers.PC, c.Read(c.Registers.PC), int8(c.Read(c.Registers.PC)))
	c.RelAddr = int8(c.Read(c.Registers.GetPC()) & 0x00FF)
	c.Registers.IncPC()
	c.cpuCycles(1)
}

// 8 bit immediate data
func R_N8(c *CPU) {
	lo := c.Read(c.Registers.GetPC())
	c.cpuCycles(1)
	c.Registers.IncPC()
	c.Fetched = uint16(lo)
}

func A8_A(c *CPU) {
	lo := uint16(c.Read(c.Registers.GetPC())) + 0xFF00
	c.cpuCycles(1)
	c.Registers.IncPC()
	c.Fetched = lo
}

func A_A8(c *CPU) {
	lo := uint16(c.Read(c.Registers.GetPC())) + 0xFF00
	c.cpuCycles(1)
	c.Registers.IncPC()
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
	return b+c > a
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

func (c *CPU) SetDecRegFlags(register uint8) {
	c.Registers.SetFlag(registers.SUBTRACTION_FLAG, true)
	c.Registers.SetFlag(registers.ZERO_FLAG, register-1 == 0)
	c.Registers.SetFlag(registers.HALF_CARRY_FLAG, register&0x000F == 0x0000)
}

func (c *CPU) SetIncRegFlags(register uint8) {
	c.Registers.SetFlag(registers.SUBTRACTION_FLAG, false)
	c.Registers.SetFlag(registers.ZERO_FLAG, register+1 == 0)
	c.Registers.SetFlag(registers.HALF_CARRY_FLAG, register&0x000F == 0x000F)
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

func (c *CPU) SetCBRotateFlags(registerVal uint8, leftOrRight string, throughCarry bool) uint8 {
	c.Registers.SetFlag(registers.SUBTRACTION_FLAG, false)
	c.Registers.SetFlag(registers.HALF_CARRY_FLAG, false)
	var oldCarry uint8
	if c.Registers.GetFlag(registers.CARRY_FLAG) {
		oldCarry = 1
	}

	switch leftOrRight {
	case "L":
		carryBit := registerVal >> 7
		if carryBit == 0 {
			c.Registers.SetFlag(registers.CARRY_FLAG, false)
		} else if carryBit == 1 {
			c.Registers.SetFlag(registers.CARRY_FLAG, true)
		}
		if throughCarry {
			registerVal = (registerVal << 1) | oldCarry
		} else {
			registerVal = (registerVal << 1) | (registerVal >> 7)
		}
	case "R":
		carryBit := registerVal & 0x0001
		if carryBit == 0 {
			c.Registers.SetFlag(registers.CARRY_FLAG, false)
		} else if carryBit == 1 {
			c.Registers.SetFlag(registers.CARRY_FLAG, true)
		}
		if throughCarry {
			registerVal = (registerVal >> 1) | (oldCarry << 7)
		} else {
			registerVal = (registerVal >> 1) | (registerVal << 7)
		}
	}
	c.Registers.SetFlag(registers.ZERO_FLAG, registerVal == 0)
	return registerVal
}

func (c *CPU) SetShiftFlags(registerVal uint8, leftOrRight string, logically bool) uint8 {
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
		registerVal <<= 1
	case "R":
		carryBit := registerVal & 0x0001
		if carryBit == 0 {
			c.Registers.SetFlag(registers.CARRY_FLAG, false)
		} else if carryBit == 1 {
			c.Registers.SetFlag(registers.CARRY_FLAG, true)
		}
		if logically {
			registerVal >>= 1
		} else {
			// 11000110 >> 1 =       01100011
			// 11000110 & 10000000 = 10000000
			registerVal = (registerVal >> 1) | (registerVal & 0x80)
		}
	}
	c.Registers.SetFlag(registers.ZERO_FLAG, registerVal == 0)
	return registerVal
}

func (c *CPU) SetSwapFlags(registerVal uint8) uint8 {
	c.Registers.SetFlag(registers.SUBTRACTION_FLAG, false)
	c.Registers.SetFlag(registers.HALF_CARRY_FLAG, false)
	c.Registers.SetFlag(registers.CARRY_FLAG, false)

	highNibble := registerVal >> 4 & 0x000F
	lowNibble := registerVal & 0x000F

	return (lowNibble << 4) | highNibble
}

func (c *CPU) SetBitFlags(registerVal uint8, bit uint8) {
	c.Registers.SetFlag(registers.SUBTRACTION_FLAG, false)
	c.Registers.SetFlag(registers.HALF_CARRY_FLAG, true)
	var i uint8
	var bitValue uint8 = 1
	for i = 0; i < bit; i++ {
		bitValue *= 2
	}
	// 00101110
	//&00010000
	// 00000000
	c.Registers.SetFlag(registers.ZERO_FLAG, registerVal&bitValue == 0)
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

func (c *CPU) SetCpFlags(a, b uint8) {
	c.Registers.SetFlag(registers.ZERO_FLAG, a-b == 0)
	c.Registers.SetFlag(registers.SUBTRACTION_FLAG, true)
	c.Registers.SetFlag(registers.HALF_CARRY_FLAG, HalfCarrySub(a, b))
	c.Registers.SetFlag(registers.CARRY_FLAG, b > a)
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

func (c *CPU) SetAndFlags(a uint8) {
	c.Registers.SetFlag(registers.ZERO_FLAG, a == 0)
	c.Registers.SetFlag(registers.SUBTRACTION_FLAG, false)
	c.Registers.SetFlag(registers.HALF_CARRY_FLAG, true)
	c.Registers.SetFlag(registers.CARRY_FLAG, false)
}

func (c *CPU) SetXorFlags(a uint8) {
	c.Registers.SetFlag(registers.ZERO_FLAG, a == 0)
	c.Registers.SetFlag(registers.SUBTRACTION_FLAG, false)
	c.Registers.SetFlag(registers.HALF_CARRY_FLAG, false)
	c.Registers.SetFlag(registers.CARRY_FLAG, false)
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
			c.Write(c.Registers.GetBC(), c.Registers.GetReg(registers.A))
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
			c.SetIncRegFlags(c.Registers.GetReg(registers.A))
			c.Registers.IncReg(registers.A)
		},
	},
	0x05: {
		Mnemonic: "DEC_B",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			c.SetDecRegFlags(c.Registers.GetReg(registers.B))
			c.Registers.DecReg(registers.B)
		},
	},
	0x06: {
		Mnemonic: "LD_B_N8",
		Size:     2,
		Ticks:    []uint8{8},
		AddrMode: R_N8,
		Operation: func(c *CPU) {
			c.Registers.SetReg(registers.B, uint8(c.Fetched))
		},
	},
	0x07: {
		Mnemonic: "RLCA",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			a := c.Registers.GetReg(registers.A)
			c.SetRotateFlags(a, "L")
			// a = 11101000
			// a << 1 = 11010000
			// a >> 7 = 00000001
			//          11010001
			c.Registers.SetReg(registers.A, (a<<1)|(a>>7))
		},
	},
	0x08: {
		Mnemonic: "LD_[A16]_SP",
		Size:     3,
		Ticks:    []uint8{20},
		AddrMode: R_A16,
		Operation: func(c *CPU) {
			lo := uint8(c.Registers.GetSP() & 0x00FF)
			hi := uint8(c.Registers.GetSP() >> 8)
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
			c.Registers.SetReg(registers.A, c.Read(c.Registers.GetBC()))
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
			c.SetIncRegFlags(c.Registers.GetReg(registers.C))
			c.Registers.IncReg(registers.C)
		},
	},
	0x0D: {
		Mnemonic: "DEC_C",
		Size:     1,
		AddrMode: NONE,
		Ticks:    []uint8{4},
		Operation: func(c *CPU) {
			c.SetDecRegFlags(c.Registers.GetReg(registers.C))
			c.Registers.DecReg(registers.C)
		},
	},
	0x0E: {
		Mnemonic: "LD_C_N8",
		Size:     2,
		Ticks:    []uint8{8},
		AddrMode: R_N8,
		Operation: func(c *CPU) {
			c.Registers.SetReg(registers.C, uint8(c.Fetched))
		},
	},
	0x0F: {
		Mnemonic: "RRCA",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			a := c.Registers.GetReg(registers.A)
			c.SetRotateFlags(a, "R")
			// a = 11101001
			// a >> 1 = 01110100
			// a << 7 = 10000000
			//          11110100
			c.Registers.SetReg(registers.A, (a>>1)|(a<<7))
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
			c.Write(c.Registers.GetDE(), c.Registers.GetReg(registers.A))
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
			c.SetIncRegFlags(c.Registers.GetReg(registers.D))
			c.Registers.IncReg(registers.D)
		},
	},
	0x15: {
		Mnemonic: "DEC_D",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			c.SetDecRegFlags(c.Registers.GetReg(registers.D))
			c.Registers.DecReg(registers.D)
		},
	},
	0x16: {
		Mnemonic: "LD_D_N8",
		Size:     2,
		Ticks:    []uint8{8},
		AddrMode: R_N8,
		Operation: func(c *CPU) {
			c.Registers.SetReg(registers.D, uint8(c.Fetched))
		},
	},
	0x17: {
		Mnemonic: "RLA",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			a := c.Registers.GetReg(registers.A)
			var oldCarry uint8
			if c.Registers.GetFlag(registers.CARRY_FLAG) {
				oldCarry = 1
			}
			c.SetRotateFlags(a, "L")
			// oldCarry = 1
			// a = 10010100
			// a << 1 = 00101000
			// a | 00000001 = 00101001
			c.Registers.SetReg(registers.A, (a<<1)|oldCarry)
		},
	},
	0x18: {
		Mnemonic: "JR_E8",
		Size:     2,
		Ticks:    []uint8{12},
		AddrMode: E8,
		Operation: func(c *CPU) {
			c.Registers.SetPC(c.Registers.GetPC() + uint16(c.RelAddr))
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
			c.Registers.SetReg(registers.A, fetched)
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
			c.SetIncRegFlags(c.Registers.GetReg(registers.E))
			c.Registers.IncReg(registers.E)
		},
	},
	0x1D: {
		Mnemonic: "DEC_E",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			c.SetDecRegFlags(c.Registers.GetReg(registers.E))
			c.Registers.DecReg(registers.E)
		},
	},
	0x1E: {
		Mnemonic: "LD_E_N8",
		Size:     2,
		Ticks:    []uint8{8},
		AddrMode: R_N8,
		Operation: func(c *CPU) {
			c.Registers.SetReg(registers.E, uint8(c.Fetched))
		},
	},
	0x1F: {
		Mnemonic: "RRA",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			a := c.Registers.GetReg(registers.A)
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
			c.Registers.SetReg(registers.A, (a>>1)|(oldCarry<<7))
		},
	},
	0x20: {
		Mnemonic: "JR_NZ_E8",
		Size:     2,
		Ticks:    []uint8{12, 8},
		AddrMode: E8,
		Operation: func(c *CPU) {
			if !c.Registers.GetFlag(registers.ZERO_FLAG) {
				c.Registers.SetPC(c.Registers.GetPC() + uint16(c.RelAddr))
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
			c.Write(hl, c.Registers.GetReg(registers.A))
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
			c.SetIncRegFlags(c.Registers.GetReg(registers.H))
			c.Registers.IncReg(registers.H)
		},
	},
	0x25: {
		Mnemonic: "DEC_H",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			c.SetDecRegFlags(c.Registers.GetReg(registers.H))
			c.Registers.DecReg(registers.H)
		},
	},
	0x26: {
		Mnemonic: "LD_H_N8",
		Size:     2,
		Ticks:    []uint8{8},
		AddrMode: R_N8,
		Operation: func(c *CPU) {
			c.Registers.SetReg(registers.H, uint8(c.Fetched))
		},
	},
	0x27: {
		Mnemonic: "DAA",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			var adj uint8 = 0
			carryFlag := false
			if c.Registers.GetFlag(registers.SUBTRACTION_FLAG) {
				if c.Registers.GetFlag(registers.HALF_CARRY_FLAG) {
					adj += 0x0006
				}
				if c.Registers.GetFlag(registers.CARRY_FLAG) {
					adj += 0x0060
				}
				c.Registers.SetReg(registers.A, c.Registers.GetReg(registers.A)-adj)
			} else {
				if c.Registers.GetFlag(registers.HALF_CARRY_FLAG) || c.Registers.GetReg(registers.A)&0x000F > 0x0009 {
					adj += 0x0006
				}
				if c.Registers.GetFlag(registers.CARRY_FLAG) || c.Registers.GetReg(registers.A) > 0x0099 {
					carryFlag = true
					adj += 0x0060
				}
				c.Registers.SetReg(registers.A, c.Registers.GetReg(registers.A)+adj)
			}
			c.Registers.SetFlag(registers.HALF_CARRY_FLAG, false)
			c.Registers.SetFlag(registers.ZERO_FLAG, c.Registers.GetReg(registers.A) == 0)
			c.Registers.SetFlag(registers.CARRY_FLAG, carryFlag)
		},
	},
	0x28: {
		Mnemonic: "JR_Z_E8",
		Size:     2,
		Ticks:    []uint8{12, 8},
		AddrMode: E8,
		Operation: func(c *CPU) {
			if c.Registers.GetFlag(registers.ZERO_FLAG) {
				c.Registers.SetPC(c.Registers.GetPC() + uint16(c.RelAddr))
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
			c.Registers.SetReg(registers.A, fetched)
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
			c.SetIncRegFlags(c.Registers.GetReg(registers.L))
			c.Registers.IncReg(registers.L)
		},
	},
	0x2D: {
		Mnemonic: "DEC_L",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			c.SetDecRegFlags(c.Registers.GetReg(registers.L))
			c.Registers.DecReg(registers.L)
		},
	},
	0x2E: {
		Mnemonic: "LD_L_N8",
		Size:     2,
		Ticks:    []uint8{8},
		AddrMode: R_N8,
		Operation: func(c *CPU) {
			c.Registers.SetReg(registers.L, uint8(c.Fetched))
		},
	},
	0x2F: {
		Mnemonic: "CPL",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			c.Registers.SetReg(registers.A, ^c.Registers.GetReg(registers.A))
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
				c.Registers.SetPC(c.Registers.GetPC() + uint16(c.RelAddr))
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
			c.Registers.SetSP(c.Fetched)
		},
	},
	0x32: {
		Mnemonic: "LD_[HLD]_A",
		Size:     1,
		Ticks:    []uint8{8},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			hl := c.Registers.GetHL()
			c.Write(hl, c.Registers.GetReg(registers.A))
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
			c.Registers.IncSP()
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
				c.Registers.SetPC(c.Registers.GetPC() + uint16(c.RelAddr))
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
			c.SetAddFlags16(hl, c.Registers.GetSP())
			c.Registers.SetHL(hl + c.Registers.GetSP())
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
			c.Registers.SetReg(registers.A, fetched)
			c.Registers.SetHL(c.Registers.GetHL() - 1)
		},
	},
	0x3B: {
		Mnemonic: "DEC_SP",
		Size:     1,
		Ticks:    []uint8{8},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			c.Registers.DecSP()
			c.cpuCycles(1)
		},
	},
	0x3C: {
		Mnemonic: "INC_A",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			c.SetIncRegFlags(c.Registers.GetReg(registers.A))
			c.Registers.IncReg(registers.A)
		},
	},
	0x3D: {
		Mnemonic: "DEC_A",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			c.SetDecRegFlags(c.Registers.GetReg(registers.A))
			c.Registers.DecReg(registers.A)
		},
	},
	0x3E: {
		Mnemonic: "LD_A_N8",
		Size:     2,
		Ticks:    []uint8{8},
		AddrMode: R_N8,
		Operation: func(c *CPU) {
			c.Registers.SetReg(registers.A, uint8(c.Fetched))
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
			c.Registers.SetReg(registers.B, c.Registers.GetReg(registers.C))
		},
	},
	0x42: {
		Mnemonic: "LD_B_D",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			c.Registers.SetReg(registers.B, c.Registers.GetReg(registers.D))
		},
	},
	0x43: {
		Mnemonic: "LD_B_E",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			c.Registers.SetReg(registers.B, c.Registers.GetReg(registers.E))
		},
	},
	0x44: {
		Mnemonic: "LD_B_H",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			c.Registers.SetReg(registers.B, c.Registers.GetReg(registers.H))
		},
	},
	0x45: {
		Mnemonic: "LD_B_L",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			c.Registers.SetReg(registers.B, c.Registers.GetReg(registers.L))
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
			c.Registers.SetReg(registers.B, val)
		},
	},
	0x47: {
		Mnemonic: "LD_B_A",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			c.Registers.SetReg(registers.B, c.Registers.GetReg(registers.A))
		},
	},
	0x48: {
		Mnemonic: "LD_C_B",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			c.Registers.SetReg(registers.C, c.Registers.GetReg(registers.B))
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
			c.Registers.SetReg(registers.C, c.Registers.GetReg(registers.D))
		},
	},
	0x4B: {
		Mnemonic: "LD_C_E",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			c.Registers.SetReg(registers.C, c.Registers.GetReg(registers.E))
		},
	},
	0x4C: {
		Mnemonic: "LD_C_H",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			c.Registers.SetReg(registers.C, c.Registers.GetReg(registers.H))
		},
	},
	0x4D: {
		Mnemonic: "LD_C_L",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			c.Registers.SetReg(registers.C, c.Registers.GetReg(registers.L))
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
			c.Registers.SetReg(registers.C, val)
		},
	},
	0x4F: {
		Mnemonic: "LD_C_A",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			c.Registers.SetReg(registers.C, c.Registers.GetReg(registers.A))
		},
	},
	0x50: {
		Mnemonic: "LD_D_B",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			c.Registers.SetReg(registers.D, c.Registers.GetReg(registers.B))
		},
	},
	0x51: {
		Mnemonic: "LD_D_C",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			c.Registers.SetReg(registers.D, c.Registers.GetReg(registers.C))
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
			c.Registers.SetReg(registers.D, c.Registers.GetReg(registers.E))
		},
	},
	0x54: {
		Mnemonic: "LD_D_H",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			c.Registers.SetReg(registers.D, c.Registers.GetReg(registers.H))
		},
	},
	0x55: {
		Mnemonic: "LD_D_L",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			c.Registers.SetReg(registers.D, c.Registers.GetReg(registers.L))
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
			c.Registers.SetReg(registers.D, val)
		},
	},
	0x57: {
		Mnemonic: "LD_D_A",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			c.Registers.SetReg(registers.D, c.Registers.GetReg(registers.A))
		},
	},
	0x58: {
		Mnemonic: "LD_E_B",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			c.Registers.SetReg(registers.E, c.Registers.GetReg(registers.B))
		},
	},
	0x59: {
		Mnemonic: "LD_E_C",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			c.Registers.SetReg(registers.E, c.Registers.GetReg(registers.C))
		},
	},
	0x5A: {
		Mnemonic: "LD_E_D",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			c.Registers.SetReg(registers.E, c.Registers.GetReg(registers.D))
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
			c.Registers.SetReg(registers.E, c.Registers.GetReg(registers.H))
		},
	},
	0x5D: {
		Mnemonic: "LD_E_L",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			c.Registers.SetReg(registers.E, c.Registers.GetReg(registers.L))
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
			c.Registers.SetReg(registers.E, val)
		},
	},
	0x5F: {
		Mnemonic: "LD_E_A",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			c.Registers.SetReg(registers.E, c.Registers.GetReg(registers.A))
		},
	},
	0x60: {
		Mnemonic: "LD_H_B",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			c.Registers.SetReg(registers.H, c.Registers.GetReg(registers.B))
		},
	},
	0x61: {
		Mnemonic: "LD_H_C",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			c.Registers.SetReg(registers.H, c.Registers.GetReg(registers.C))
		},
	},
	0x62: {
		Mnemonic: "LD_H_D",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			c.Registers.SetReg(registers.H, c.Registers.GetReg(registers.D))
		},
	},
	0x63: {
		Mnemonic: "LD_H_E",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			c.Registers.SetReg(registers.H, c.Registers.GetReg(registers.E))
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
			c.Registers.SetReg(registers.H, c.Registers.GetReg(registers.L))
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
			c.Registers.SetReg(registers.H, val)
		},
	},
	0x67: {
		Mnemonic: "LD_H_A",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			c.Registers.SetReg(registers.H, c.Registers.GetReg(registers.A))
		},
	},
	0x68: {
		Mnemonic: "LD_L_B",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			c.Registers.SetReg(registers.L, c.Registers.GetReg(registers.B))
		},
	},
	0x69: {
		Mnemonic: "LD_L_C",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			c.Registers.SetReg(registers.L, c.Registers.GetReg(registers.C))
		},
	},
	0x6A: {
		Mnemonic: "LD_L_D",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			c.Registers.SetReg(registers.L, c.Registers.GetReg(registers.D))
		},
	},
	0x6B: {
		Mnemonic: "LD_L_E",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			c.Registers.SetReg(registers.L, c.Registers.GetReg(registers.E))
		},
	},
	0x6C: {
		Mnemonic: "LD_L_H",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			c.Registers.SetReg(registers.L, c.Registers.GetReg(registers.H))
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
			c.Registers.SetReg(registers.L, val)
		},
	},
	0x6F: {
		Mnemonic: "LD_L_A",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			c.Registers.SetReg(registers.L, c.Registers.GetReg(registers.A))
		},
	},
	0x70: {
		Mnemonic: "LD_[HL]_B",
		Size:     1,
		Ticks:    []uint8{8},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			c.Write(c.Registers.GetHL(), c.Registers.GetReg(registers.B))
			c.cpuCycles(1)
		},
	},
	0x71: {
		Mnemonic: "LD_[HL]_C",
		Size:     1,
		Ticks:    []uint8{8},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			c.Write(c.Registers.GetHL(), c.Registers.GetReg(registers.C))
			c.cpuCycles(1)
		},
	},
	0x72: {
		Mnemonic: "LD_[HL]_D",
		Size:     1,
		Ticks:    []uint8{8},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			c.Write(c.Registers.GetHL(), c.Registers.GetReg(registers.D))
			c.cpuCycles(1)
		},
	},
	0x73: {
		Mnemonic: "LD_[HL]_E",
		Size:     1,
		Ticks:    []uint8{8},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			c.Write(c.Registers.GetHL(), c.Registers.GetReg(registers.E))
			c.cpuCycles(1)
		},
	},
	0x74: {
		Mnemonic: "LD_[HL]_H",
		Size:     1,
		Ticks:    []uint8{8},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			c.Write(c.Registers.GetHL(), c.Registers.GetReg(registers.H))
			c.cpuCycles(1)
		},
	},
	0x75: {
		Mnemonic: "LD_[HL]_L",
		Size:     1,
		Ticks:    []uint8{8},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			c.Write(c.Registers.GetHL(), c.Registers.GetReg(registers.L))
			c.cpuCycles(1)
		},
	},
	0x76: {
		Mnemonic: "HALT",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			c.Halted = true
		},
	},
	0x77: {
		Mnemonic: "LD_[HL]_A",
		Size:     1,
		Ticks:    []uint8{8},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			c.Write(c.Registers.GetHL(), c.Registers.GetReg(registers.A))
			c.cpuCycles(1)
		},
	},
	0x78: {
		Mnemonic: "LD_A_B",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			c.Registers.SetReg(registers.A, c.Registers.GetReg(registers.B))
		},
	},
	0x79: {
		Mnemonic: "LD_A_C",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			c.Registers.SetReg(registers.A, c.Registers.GetReg(registers.C))
		},
	},
	0x7A: {
		Mnemonic: "LD_A_D",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			c.Registers.SetReg(registers.A, c.Registers.GetReg(registers.D))
		},
	},
	0x7B: {
		Mnemonic: "LD_A_E",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			c.Registers.SetReg(registers.A, c.Registers.GetReg(registers.E))
		},
	},
	0x7C: {
		Mnemonic: "LD_A_H",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			c.Registers.SetReg(registers.A, c.Registers.GetReg(registers.H))
		},
	},
	0x7D: {
		Mnemonic: "LD_A_L",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			c.Registers.SetReg(registers.A, c.Registers.GetReg(registers.L))
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
			c.Registers.SetReg(registers.A, val)
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
			a := c.Registers.GetReg(registers.A)
			c.SetAddFlags(a, c.Registers.GetReg(registers.B))
			c.Registers.SetReg(registers.A, a+c.Registers.GetReg(registers.B))
		},
	},
	0x81: {
		Mnemonic: "ADD_A_C",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			a := c.Registers.GetReg(registers.A)
			c.SetAddFlags(a, c.Registers.GetReg(registers.C))
			c.Registers.SetReg(registers.A, a+c.Registers.GetReg(registers.C))
		},
	},
	0x82: {
		Mnemonic: "ADD_A_D",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			a := c.Registers.GetReg(registers.A)
			c.SetAddFlags(a, c.Registers.GetReg(registers.D))
			c.Registers.SetReg(registers.A, a+c.Registers.GetReg(registers.D))
		},
	},
	0x83: {
		Mnemonic: "ADD_A_E",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			a := c.Registers.GetReg(registers.A)
			c.SetAddFlags(a, c.Registers.GetReg(registers.E))
			c.Registers.SetReg(registers.A, a+c.Registers.GetReg(registers.E))
		},
	},
	0x84: {
		Mnemonic: "ADD_A_H",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			a := c.Registers.GetReg(registers.A)
			c.SetAddFlags(a, c.Registers.GetReg(registers.H))
			c.Registers.SetReg(registers.A, a+c.Registers.GetReg(registers.H))
		},
	},
	0x85: {
		Mnemonic: "ADD_A_L",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			a := c.Registers.GetReg(registers.A)
			c.SetAddFlags(a, c.Registers.GetReg(registers.L))
			c.Registers.SetReg(registers.A, a+c.Registers.GetReg(registers.L))
		},
	},
	0x86: {
		Mnemonic: "ADD_A_[HL]",
		Size:     1,
		Ticks:    []uint8{8},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			val := c.Read(c.Registers.GetHL())
			a := c.Registers.GetReg(registers.A)
			c.SetAddFlags(a, val)
			c.Registers.SetReg(registers.A, a+val)
		},
	},
	0x87: {
		Mnemonic: "ADD_A_A",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			a := c.Registers.GetReg(registers.A)
			c.SetAddFlags(a, a)
			c.Registers.SetReg(registers.A, a+a)
		},
	},
	0x88: {
		Mnemonic: "ADC_A_B",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			a := c.Registers.GetReg(registers.A)
			carryFlag := c.SetAdcFlags(a, c.Registers.GetReg(registers.B))
			c.Registers.SetReg(registers.A, a+(c.Registers.GetReg(registers.B)+carryFlag))
		},
	},
	0x89: {
		Mnemonic: "ADC_A_C",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			a := c.Registers.GetReg(registers.A)
			carryFlag := c.SetAdcFlags(a, c.Registers.GetReg(registers.C))
			c.Registers.SetReg(registers.A, a+(c.Registers.GetReg(registers.C)+carryFlag))
		},
	},
	0x8A: {
		Mnemonic: "ADC_A_D",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			a := c.Registers.GetReg(registers.A)
			carryFlag := c.SetAdcFlags(a, c.Registers.GetReg(registers.D))
			c.Registers.SetReg(registers.A, a+(c.Registers.GetReg(registers.D)+carryFlag))
		},
	},
	0x8B: {
		Mnemonic: "ADC_A_E",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			a := c.Registers.GetReg(registers.A)
			carryFlag := c.SetAdcFlags(a, c.Registers.GetReg(registers.E))
			c.Registers.SetReg(registers.A, a+(c.Registers.GetReg(registers.E)+carryFlag))
		},
	},
	0x8C: {
		Mnemonic: "ADC_A_H",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			a := c.Registers.GetReg(registers.A)
			carryFlag := c.SetAdcFlags(a, c.Registers.GetReg(registers.H))
			c.Registers.SetReg(registers.A, a+(c.Registers.GetReg(registers.H)+carryFlag))
		},
	},
	0x8D: {
		Mnemonic: "ADC_A_L",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			a := c.Registers.GetReg(registers.A)
			carryFlag := c.SetAdcFlags(a, c.Registers.GetReg(registers.L))
			c.Registers.SetReg(registers.A, a+(c.Registers.GetReg(registers.L)+carryFlag))
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
			a := c.Registers.GetReg(registers.A)
			carryFlag := c.SetAdcFlags(a, val)
			c.Registers.SetReg(registers.A, a+(val+carryFlag))
		},
	},
	0x8F: {
		Mnemonic: "ADC_A_A",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			a := c.Registers.GetReg(registers.A)
			carryFlag := c.SetAdcFlags(a, a)
			c.Registers.SetReg(registers.A, a+(a+carryFlag))
		},
	},
	0x90: {
		Mnemonic: "SUB_A_B",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			a := c.Registers.GetReg(registers.A)
			c.SetSubFlags(a, c.Registers.GetReg(registers.B))
			c.Registers.SetReg(registers.A, a-c.Registers.GetReg(registers.B))
		},
	},
	0x91: {
		Mnemonic: "SUB_A_C",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			a := c.Registers.GetReg(registers.A)
			c.SetSubFlags(a, c.Registers.GetReg(registers.C))
			c.Registers.SetReg(registers.A, a-c.Registers.GetReg(registers.C))
		},
	},
	0x92: {
		Mnemonic: "SUB_A_D",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			a := c.Registers.GetReg(registers.A)
			c.SetSubFlags(a, c.Registers.GetReg(registers.D))
			c.Registers.SetReg(registers.A, a-c.Registers.GetReg(registers.D))
		},
	},
	0x93: {
		Mnemonic: "SUB_A_E",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			a := c.Registers.GetReg(registers.A)
			c.SetSubFlags(a, c.Registers.GetReg(registers.E))
			c.Registers.SetReg(registers.A, a-c.Registers.GetReg(registers.E))
		},
	},
	0x94: {
		Mnemonic: "SUB_A_H",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			a := c.Registers.GetReg(registers.A)
			c.SetSubFlags(a, c.Registers.GetReg(registers.H))
			c.Registers.SetReg(registers.A, a-c.Registers.GetReg(registers.H))
		},
	},
	0x95: {
		Mnemonic: "SUB_A_L",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			a := c.Registers.GetReg(registers.A)
			c.SetSubFlags(a, c.Registers.GetReg(registers.L))
			c.Registers.SetReg(registers.A, a-c.Registers.GetReg(registers.L))
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
			a := c.Registers.GetReg(registers.A)
			c.SetSubFlags(a, val)
			c.Registers.SetReg(registers.A, a-val)
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
			c.Registers.SetReg(registers.A, 0)
		},
	},
	0x98: {
		Mnemonic: "SBC_A_B",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			a := c.Registers.GetReg(registers.A)
			carryFlag := c.SetSbcFlags(a, c.Registers.GetReg(registers.B))
			c.Registers.SetReg(registers.A, a-(c.Registers.GetReg(registers.B)-carryFlag))
		},
	},
	0x99: {
		Mnemonic: "SBC_A_C",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			a := c.Registers.GetReg(registers.A)
			carryFlag := c.SetSbcFlags(a, c.Registers.GetReg(registers.C))
			c.Registers.SetReg(registers.A, a-(c.Registers.GetReg(registers.C)-carryFlag))
		},
	},
	0x9A: {
		Mnemonic: "SBC_A_D",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			a := c.Registers.GetReg(registers.A)
			carryFlag := c.SetSbcFlags(a, c.Registers.GetReg(registers.D))
			c.Registers.SetReg(registers.A, a-(c.Registers.GetReg(registers.D)-carryFlag))
		},
	},
	0x9B: {
		Mnemonic: "SBC_A_E",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			a := c.Registers.GetReg(registers.A)
			carryFlag := c.SetSbcFlags(a, c.Registers.GetReg(registers.E))
			c.Registers.SetReg(registers.A, a-(c.Registers.GetReg(registers.E)-carryFlag))
		},
	},
	0x9C: {
		Mnemonic: "SBC_A_H",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			a := c.Registers.GetReg(registers.A)
			carryFlag := c.SetSbcFlags(a, c.Registers.GetReg(registers.H))
			c.Registers.SetReg(registers.A, a-(c.Registers.GetReg(registers.H)-carryFlag))
		},
	},
	0x9D: {
		Mnemonic: "SBC_A_L",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			a := c.Registers.GetReg(registers.A)
			carryFlag := c.SetSbcFlags(a, c.Registers.GetReg(registers.L))
			c.Registers.SetReg(registers.A, a-(c.Registers.GetReg(registers.L)-carryFlag))
		},
	},
	0x9E: {
		Mnemonic: "SBC_A_[HL]",
		Size:     1,
		Ticks:    []uint8{8},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			val := c.Read(c.Registers.GetHL())
			c.cpuCycles(1)
			a := c.Registers.GetReg(registers.A)
			carryFlag := c.SetSbcFlags(a, val)
			c.Registers.SetReg(registers.A, a-(val+carryFlag))
		},
	},
	0x9F: {
		Mnemonic: "SBC_A_A",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			a := c.Registers.GetReg(registers.A)
			oldCarryFlag := c.Registers.GetFlag(registers.CARRY_FLAG)
			carryFlag := c.SetSbcFlags(a, a)
			c.Registers.SetReg(registers.A, a-(a-carryFlag))
			// set carryFlag back to original value as it should not be affected by this opcode
			c.Registers.SetFlag(registers.SUBTRACTION_FLAG, oldCarryFlag)
		},
	},
	0xA0: {
		Mnemonic: "AND_A_B",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			a := c.Registers.GetReg(registers.A)
			c.Registers.SetReg(registers.A, a&c.Registers.GetReg(registers.B))
			c.SetAndFlags(c.Registers.GetReg(registers.A))
		},
	},
	0xA1: {
		Mnemonic: "AND_A_C",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			a := c.Registers.GetReg(registers.A)
			c.Registers.SetReg(registers.A, a&c.Registers.GetReg(registers.C))
			c.SetAndFlags(c.Registers.GetReg(registers.A))
		},
	},
	0xA2: {
		Mnemonic: "AND_A_D",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			a := c.Registers.GetReg(registers.A)
			c.Registers.SetReg(registers.A, a&c.Registers.GetReg(registers.D))
			c.SetAndFlags(c.Registers.GetReg(registers.A))
		},
	},
	0xA3: {
		Mnemonic: "AND_A_E",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			a := c.Registers.GetReg(registers.A)
			c.Registers.SetReg(registers.A, a&c.Registers.GetReg(registers.E))
			c.SetAndFlags(c.Registers.GetReg(registers.A))
		},
	},
	0xA4: {
		Mnemonic: "AND_A_H",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			a := c.Registers.GetReg(registers.A)
			c.Registers.SetReg(registers.A, a&c.Registers.GetReg(registers.H))
			c.SetAndFlags(c.Registers.GetReg(registers.A))
		},
	},
	0xA5: {
		Mnemonic: "AND_A_L",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			a := c.Registers.GetReg(registers.A)
			c.Registers.SetReg(registers.A, a&c.Registers.GetReg(registers.L))
			c.SetAndFlags(c.Registers.GetReg(registers.A))
		},
	},
	0xA6: {
		Mnemonic: "AND_A_[HL]",
		Size:     1,
		Ticks:    []uint8{8},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			val := c.Read(c.Registers.GetHL())
			c.cpuCycles(1)
			a := c.Registers.GetReg(registers.A)
			c.Registers.SetReg(registers.A, a&val)
			c.SetAndFlags(c.Registers.GetReg(registers.A))
		},
	},
	0xA7: {
		Mnemonic: "AND_A_A",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			a := c.Registers.GetReg(registers.A)
			c.Registers.SetReg(registers.A, a&c.Registers.GetReg(registers.A))
			c.SetAndFlags(c.Registers.GetReg(registers.A))
		},
	},
	0xA8: {
		Mnemonic: "XOR_A_B",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			a := c.Registers.GetReg(registers.A)
			c.Registers.SetReg(registers.A, a^c.Registers.GetReg(registers.B))
			c.SetXorFlags(c.Registers.GetReg(registers.A))
		},
	},
	0xA9: {
		Mnemonic: "XOR_A_C",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			a := c.Registers.GetReg(registers.A)
			c.Registers.SetReg(registers.A, a^c.Registers.GetReg(registers.C))
			c.SetXorFlags(c.Registers.GetReg(registers.A))
		},
	},
	0xAA: {
		Mnemonic: "XOR_A_D",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			a := c.Registers.GetReg(registers.A)
			c.Registers.SetReg(registers.A, a^c.Registers.GetReg(registers.D))
			c.SetXorFlags(c.Registers.GetReg(registers.A))
		},
	},
	0xAB: {
		Mnemonic: "XOR_A_E",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			a := c.Registers.GetReg(registers.A)
			c.Registers.SetReg(registers.A, a^c.Registers.GetReg(registers.E))
			c.SetXorFlags(c.Registers.GetReg(registers.A))
		},
	},
	0xAC: {
		Mnemonic: "XOR_A_H",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			a := c.Registers.GetReg(registers.A)
			c.Registers.SetReg(registers.A, a^c.Registers.GetReg(registers.H))
			c.SetXorFlags(c.Registers.GetReg(registers.A))
		},
	},
	0xAD: {
		Mnemonic: "XOR_A_L",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			a := c.Registers.GetReg(registers.A)
			c.Registers.SetReg(registers.A, a^c.Registers.GetReg(registers.L))
			c.SetXorFlags(c.Registers.GetReg(registers.A))
		},
	},
	0xAE: {
		Mnemonic: "XOR_A_[HL]",
		Size:     1,
		Ticks:    []uint8{8},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			val := c.Read(c.Registers.GetHL())
			c.cpuCycles(1)
			a := c.Registers.GetReg(registers.A)
			c.Registers.SetReg(registers.A, a^val)
			c.SetXorFlags(c.Registers.GetReg(registers.A))
		},
	},
	0xAF: {
		Mnemonic: "XOR_A_A",
		Size:     1,
		AddrMode: NONE,
		Operation: func(c *CPU) {
			a := c.Registers.GetReg(registers.A)
			c.Registers.SetReg(registers.A, a^c.Registers.GetReg(registers.A))
			c.SetXorFlags(c.Registers.GetReg(registers.A))
		},
	},
	0xB0: {
		Mnemonic: "OR_A_B",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			a := c.Registers.GetReg(registers.A)
			c.Registers.SetReg(registers.A, a|c.Registers.GetReg(registers.B))
			c.SetXorFlags(c.Registers.GetReg(registers.A))
		},
	},
	0xB1: {
		Mnemonic: "OR_A_C",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			a := c.Registers.GetReg(registers.A)
			c.Registers.SetReg(registers.A, a|c.Registers.GetReg(registers.C))
			c.SetXorFlags(c.Registers.GetReg(registers.A))
		},
	},
	0xB2: {
		Mnemonic: "OR_A_D",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			a := c.Registers.GetReg(registers.A)
			c.Registers.SetReg(registers.A, a|c.Registers.GetReg(registers.D))
			c.SetXorFlags(c.Registers.GetReg(registers.A))
		},
	},
	0xB3: {
		Mnemonic: "OR_A_E",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			a := c.Registers.GetReg(registers.A)
			c.Registers.SetReg(registers.A, a|c.Registers.GetReg(registers.E))
			c.SetXorFlags(c.Registers.GetReg(registers.A))
		},
	},
	0xB4: {
		Mnemonic: "OR_A_H",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			a := c.Registers.GetReg(registers.A)
			c.Registers.SetReg(registers.A, a|c.Registers.GetReg(registers.H))
			c.SetXorFlags(c.Registers.GetReg(registers.A))
		},
	},
	0xB5: {
		Mnemonic: "OR_A_L",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			a := c.Registers.GetReg(registers.A)
			c.Registers.SetReg(registers.A, a|c.Registers.GetReg(registers.L))
			c.SetXorFlags(c.Registers.GetReg(registers.A))
		},
	},
	0xB6: {
		Mnemonic: "OR_A_[HL]",
		Size:     1,
		Ticks:    []uint8{8},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			val := c.Read(c.Registers.GetHL())
			c.cpuCycles(1)
			a := c.Registers.GetReg(registers.A)
			c.Registers.SetReg(registers.A, a|val)
			c.SetXorFlags(c.Registers.GetReg(registers.A))
		},
	},
	0xB7: {
		Mnemonic: "OR_A_A",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			a := c.Registers.GetReg(registers.A)
			c.Registers.SetReg(registers.A, a|a)
			c.SetXorFlags(c.Registers.GetReg(registers.A))
		},
	},
	0xB8: {
		Mnemonic: "CP_A_B",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			c.SetCpFlags(c.Registers.GetReg(registers.A), c.Registers.GetReg(registers.B))
		},
	},
	0xB9: {
		Mnemonic: "CP_A_C",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			c.SetCpFlags(c.Registers.GetReg(registers.A), c.Registers.GetReg(registers.C))
		},
	},
	0xBA: {
		Mnemonic: "CP_A_D",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			c.SetCpFlags(c.Registers.GetReg(registers.A), c.Registers.GetReg(registers.D))
		},
	},
	0xBB: {
		Mnemonic: "CP_A_E",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			c.SetCpFlags(c.Registers.GetReg(registers.A), c.Registers.GetReg(registers.E))
		},
	},
	0xBC: {
		Mnemonic: "CP_A_H",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			c.SetCpFlags(c.Registers.GetReg(registers.A), c.Registers.GetReg(registers.H))
		},
	},
	0xBD: {
		Mnemonic: "CP_A_L",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			c.SetCpFlags(c.Registers.GetReg(registers.A), c.Registers.GetReg(registers.L))
		},
	},
	0xBE: {
		Mnemonic: "CP_A_[HL]",
		Size:     1,
		Ticks:    []uint8{8},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			val := c.Read(c.Registers.GetHL())
			c.cpuCycles(1)
			c.SetCpFlags(c.Registers.GetReg(registers.A), val)
		},
	},
	0xBF: {
		Mnemonic: "CP_A_A",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			c.SetCpFlags(c.Registers.GetReg(registers.A), c.Registers.GetReg(registers.A))
		},
	},
	0xC0: {
		Mnemonic: "RET_NZ",
		Size:     1,
		Ticks:    []uint8{20, 8},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			if !c.Registers.GetFlag(registers.ZERO_FLAG) {
				val := c.StackPop16()
				c.Registers.SetPC(val)
				c.cpuCycles(1)
			}
			c.cpuCycles(1)
		},
	},
	0xC1: {
		Mnemonic: "POP_BC",
		Size:     1,
		Ticks:    []uint8{12},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			val := c.StackPop16()
			c.Registers.SetBC(val)
			c.cpuCycles(1)
		},
	},
	0xC2: {
		Mnemonic: "JP_NZ_A16",
		Size:     3,
		Ticks:    []uint8{16, 12},
		AddrMode: R_A16,
		Operation: func(c *CPU) {
			if !c.Registers.GetFlag(registers.ZERO_FLAG) {
				c.Registers.SetPC(c.Fetched)
				c.cpuCycles(1)
			}
		},
	},
	0xC3: {
		Mnemonic: "JP_A16",
		Size:     3,
		Ticks:    []uint8{12},
		AddrMode: R_A16,
		Operation: func(c *CPU) {
			c.Registers.SetPC(c.Fetched)
			c.cpuCycles(1)
		},
	},
	0xC4: {
		Mnemonic: "CALL_NZ_A16",
		Size:     3,
		Ticks:    []uint8{24, 12},
		AddrMode: R_A16,
		Operation: func(c *CPU) {
			if !c.Registers.GetFlag(registers.ZERO_FLAG) {
				c.StackPush16(c.Registers.GetPC())
				c.Registers.SetPC(c.Fetched)
				c.cpuCycles(1)
			}
		},
	},
	0xC5: {
		Mnemonic: "PUSH_BC",
		Size:     1,
		Ticks:    []uint8{16},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			c.StackPush16(c.Registers.GetBC())
			c.cpuCycles(1)
		},
	},
	0xC6: {
		Mnemonic: "ADD_A_N8",
		Size:     2,
		Ticks:    []uint8{8},
		AddrMode: R_N8,
		Operation: func(c *CPU) {
			a := c.Registers.GetReg(registers.A)
			c.SetAddFlags(a, uint8(c.Fetched))
			c.Registers.SetReg(registers.A, a+uint8(c.Fetched))
		},
	},
	0xC7: {
		Mnemonic: "RST_$00",
		Size:     3,
		Ticks:    []uint8{16},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			c.StackPush16(c.Registers.GetPC())
			c.Registers.SetPC(0x0000)
			c.cpuCycles(1)
		},
	},
	0xC8: {
		Mnemonic: "RET_Z",
		Size:     1,
		Ticks:    []uint8{20, 8},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			if c.Registers.GetFlag(registers.ZERO_FLAG) {
				val := c.StackPop16()
				c.Registers.SetPC(val)
				c.cpuCycles(1)
			}
			c.cpuCycles(1)
		},
	},
	0xC9: {
		Mnemonic: "RET",
		Size:     1,
		Ticks:    []uint8{16},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			val := c.StackPop16()
			c.Registers.SetPC(val)
			c.cpuCycles(1)
		},
	},
	0xCA: {
		Mnemonic: "JP_Z_A16",
		Size:     3,
		Ticks:    []uint8{16, 12},
		AddrMode: R_A16,
		Operation: func(c *CPU) {
			if c.Registers.GetFlag(registers.ZERO_FLAG) {
				c.Registers.SetPC(c.Fetched)
				c.cpuCycles(1)
			}
			c.cpuCycles(1)
		},
	},
	0xCB: {
		Mnemonic: "CB",
		Size:     3,
		Ticks:    []uint8{8, 16},
		AddrMode: R_N8,
		Operation: func(c *CPU) {
			// 0000000010001101
			firstNibble := (c.Fetched >> 4) & 0x000F
			secondNibble := c.Fetched & 0x000F
			c.cpuCycles(1)
			switch firstNibble {
			// rlc rrc
			case 0x0:
				if secondNibble < 0x8 {
					// rlc
					if secondNibble != 0x6 {
						reg, val := c.CBLookUp(uint8(secondNibble))
						c.Registers.SetReg(reg, c.SetCBRotateFlags(val, "L", false))
					} else {
						val := c.Read(c.Registers.GetHL())
						c.cpuCycles(1)
						c.Write(c.Registers.GetHL(), c.SetCBRotateFlags(val, "L", false))
						c.cpuCycles(1)
					}
				} else {
					// rrc
					if secondNibble != 0xE {
						reg, val := c.CBLookUp(uint8(secondNibble))
						c.Registers.SetReg(reg, c.SetCBRotateFlags(val, "R", false))
					} else {
						val := c.Read(c.Registers.GetHL())
						c.cpuCycles(1)
						c.Write(c.Registers.GetHL(), c.SetCBRotateFlags(val, "R", false))
						c.cpuCycles(1)
					}
				}
			// rl rr
			case 0x1:
				if secondNibble < 0x8 {
					// rlc
					if secondNibble != 0x6 {
						reg, val := c.CBLookUp(uint8(secondNibble))
						c.Registers.SetReg(reg, c.SetCBRotateFlags(val, "L", true))
					} else {
						val := c.Read(c.Registers.GetHL())
						c.cpuCycles(1)
						c.Write(c.Registers.GetHL(), c.SetCBRotateFlags(val, "L", true))
						c.cpuCycles(1)
					}
				} else {
					// rrc
					if secondNibble != 0xE {
						reg, val := c.CBLookUp(uint8(secondNibble))
						c.Registers.SetReg(reg, c.SetCBRotateFlags(val, "R", true))
					} else {
						val := c.Read(c.Registers.GetHL())
						c.cpuCycles(1)
						c.Write(c.Registers.GetHL(), c.SetCBRotateFlags(val, "R", true))
						c.cpuCycles(1)
					}
				}
			// sla sra
			case 0x2:
				if secondNibble < 0x8 {
					// sla
					if secondNibble != 0x6 {
						reg, val := c.CBLookUp(uint8(secondNibble))
						c.Registers.SetReg(reg, c.SetShiftFlags(val, "L", false))
					} else {
						val := c.Read(c.Registers.GetHL())
						c.cpuCycles(1)
						c.Write(c.Registers.GetHL(), c.SetShiftFlags(val, "L", false))
						c.cpuCycles(1)
					}
				} else {
					// sra
					if secondNibble != 0xE {
						reg, val := c.CBLookUp(uint8(secondNibble))
						c.Registers.SetReg(reg, c.SetShiftFlags(val, "R", false))
					} else {
						val := c.Read(c.Registers.GetHL())
						c.cpuCycles(1)
						c.Write(c.Registers.GetHL(), c.SetShiftFlags(val, "R", false))
						c.cpuCycles(1)
					}
				}
			// swap srl
			case 0x3:
				if secondNibble < 0x8 {
					// swap
					if secondNibble != 0x6 {
						reg, val := c.CBLookUp(uint8(secondNibble))
						c.Registers.SetReg(reg, c.SetSwapFlags(val))
					} else {
						val := c.Read(c.Registers.GetHL())
						c.cpuCycles(1)
						c.Write(c.Registers.GetHL(), c.SetSwapFlags(val))
						c.cpuCycles(1)
					}
				} else {
					// srl
					if secondNibble != 0xE {
						reg, val := c.CBLookUp(uint8(secondNibble))
						c.Registers.SetReg(reg, c.SetShiftFlags(val, "R", true))
					} else {
						val := c.Read(c.Registers.GetHL())
						c.cpuCycles(1)
						c.Write(c.Registers.GetHL(), c.SetShiftFlags(val, "R", true))
						c.cpuCycles(1)
					}
				}
			// BIT 0 BIT 1
			case 0x4:
				if secondNibble < 0x8 {
					// bit 0
					if secondNibble != 0x6 {
						_, val := c.CBLookUp(uint8(secondNibble))
						c.SetBitFlags(val, 0)
					} else {
						val := c.Read(c.Registers.GetHL())
						c.cpuCycles(1)
						c.SetBitFlags(val, 0)
					}
				} else {
					// bit 1
					if secondNibble != 0xE {
						_, val := c.CBLookUp(uint8(secondNibble))
						c.SetBitFlags(val, 1)
					} else {
						val := c.Read(c.Registers.GetHL())
						c.cpuCycles(1)
						c.SetBitFlags(val, 1)
					}
				}
			// BIT 2 BIT 3
			case 0x5:
				if secondNibble < 0x8 {
					// bit 2
					if secondNibble != 0x6 {
						_, val := c.CBLookUp(uint8(secondNibble))
						c.SetBitFlags(val, 2)
					} else {
						val := c.Read(c.Registers.GetHL())
						c.cpuCycles(1)
						c.SetBitFlags(val, 2)
					}
				} else {
					// bit 3
					if secondNibble != 0xE {
						_, val := c.CBLookUp(uint8(secondNibble))
						c.SetBitFlags(val, 3)
					} else {
						val := c.Read(c.Registers.GetHL())
						c.cpuCycles(1)
						c.SetBitFlags(val, 3)
					}
				}
			// BIT 4 BIT 5
			case 0x6:
				if secondNibble < 0x8 {
					// bit 4
					if secondNibble != 0x6 {
						_, val := c.CBLookUp(uint8(secondNibble))
						c.SetBitFlags(val, 4)
					} else {
						val := c.Read(c.Registers.GetHL())
						c.cpuCycles(1)
						c.SetBitFlags(val, 4)
					}
				} else {
					// bit 5
					if secondNibble != 0xE {
						_, val := c.CBLookUp(uint8(secondNibble))
						c.SetBitFlags(val, 5)
					} else {
						val := c.Read(c.Registers.GetHL())
						c.cpuCycles(1)
						c.SetBitFlags(val, 5)
					}
				}
			// BIT 6 BIT 7
			case 0x7:
				if secondNibble < 0x8 {
					// bit 6
					if secondNibble != 0x6 {
						_, val := c.CBLookUp(uint8(secondNibble))
						c.SetBitFlags(val, 6)
					} else {
						val := c.Read(c.Registers.GetHL())
						c.cpuCycles(1)
						c.SetBitFlags(val, 6)
					}
				} else {
					// bit 7
					if secondNibble != 0xE {
						_, val := c.CBLookUp(uint8(secondNibble))
						c.SetBitFlags(val, 7)
					} else {
						val := c.Read(c.Registers.GetHL())
						c.cpuCycles(1)
						c.SetBitFlags(val, 7)
					}
				}
			// RES 0 RES 1
			case 0x8:
				if secondNibble < 0x8 {
					// RES 0
					if secondNibble != 0x6 {
						reg, val := c.CBLookUp(uint8(secondNibble))
						c.Registers.SetReg(reg, c.SetBit(val, 0, false))
					} else {
						val := c.Read(c.Registers.GetHL())
						c.cpuCycles(1)
						val = c.SetBit(val, 0, false)
						c.Write(c.Registers.GetHL(), val)
						c.cpuCycles(1)
					}
				} else {
					// RES 1
					if secondNibble != 0xE {
						reg, val := c.CBLookUp(uint8(secondNibble))
						c.Registers.SetReg(reg, c.SetBit(val, 1, false))
					} else {
						val := c.Read(c.Registers.GetHL())
						c.cpuCycles(1)
						val = c.SetBit(val, 1, false)
						c.Write(c.Registers.GetHL(), val)
						c.cpuCycles(1)
					}
				}
			// RES 2 RES 3
			case 0x9:
				if secondNibble < 0x8 {
					// RES 2
					if secondNibble != 0x6 {
						reg, val := c.CBLookUp(uint8(secondNibble))
						c.Registers.SetReg(reg, c.SetBit(val, 2, false))
					} else {
						val := c.Read(c.Registers.GetHL())
						c.cpuCycles(1)
						val = c.SetBit(val, 2, false)
						c.Write(c.Registers.GetHL(), val)
						c.cpuCycles(1)
					}
				} else {
					// RES 3
					if secondNibble != 0xE {
						reg, val := c.CBLookUp(uint8(secondNibble))
						c.Registers.SetReg(reg, c.SetBit(val, 3, false))
					} else {
						val := c.Read(c.Registers.GetHL())
						c.cpuCycles(1)
						val = c.SetBit(val, 3, false)
						c.Write(c.Registers.GetHL(), val)
						c.cpuCycles(1)
					}
				}
			// RES 4 RES 5
			case 0xA:
				if secondNibble < 0x8 {
					// RES 4
					if secondNibble != 0x6 {
						reg, val := c.CBLookUp(uint8(secondNibble))
						c.Registers.SetReg(reg, c.SetBit(val, 4, false))
					} else {
						val := c.Read(c.Registers.GetHL())
						c.cpuCycles(1)
						val = c.SetBit(val, 4, false)
						c.Write(c.Registers.GetHL(), val)
						c.cpuCycles(1)
					}
				} else {
					// RES 5
					if secondNibble != 0xE {
						reg, val := c.CBLookUp(uint8(secondNibble))
						c.Registers.SetReg(reg, c.SetBit(val, 5, false))
					} else {
						val := c.Read(c.Registers.GetHL())
						c.cpuCycles(1)
						val = c.SetBit(val, 5, false)
						c.Write(c.Registers.GetHL(), val)
						c.cpuCycles(1)
					}
				}
			// RES 6 RES 7
			case 0xB:
				if secondNibble < 0x8 {
					// RES 6
					if secondNibble != 0x6 {
						reg, val := c.CBLookUp(uint8(secondNibble))
						c.Registers.SetReg(reg, c.SetBit(val, 6, false))
					} else {
						val := c.Read(c.Registers.GetHL())
						c.cpuCycles(1)
						val = c.SetBit(val, 6, false)
						c.Write(c.Registers.GetHL(), val)
						c.cpuCycles(1)
					}
				} else {
					// RES 7
					if secondNibble != 0xE {
						reg, val := c.CBLookUp(uint8(secondNibble))
						c.Registers.SetReg(reg, c.SetBit(val, 7, false))
					} else {
						val := c.Read(c.Registers.GetHL())
						c.cpuCycles(1)
						val = c.SetBit(val, 7, false)
						c.Write(c.Registers.GetHL(), val)
						c.cpuCycles(1)
					}
				}
			// SET 0 SET 1
			case 0xC:
				if secondNibble < 0x8 {
					// SET 0
					if secondNibble != 0x6 {
						reg, val := c.CBLookUp(uint8(secondNibble))
						c.Registers.SetReg(reg, c.SetBit(val, 0, true))
					} else {
						val := c.Read(c.Registers.GetHL())
						c.cpuCycles(1)
						val = c.SetBit(val, 0, true)
						c.Write(c.Registers.GetHL(), val)
						c.cpuCycles(1)
					}
				} else {
					// SET 1
					if secondNibble != 0xE {
						reg, val := c.CBLookUp(uint8(secondNibble))
						c.Registers.SetReg(reg, c.SetBit(val, 1, true))
					} else {
						val := c.Read(c.Registers.GetHL())
						c.cpuCycles(1)
						val = c.SetBit(val, 1, true)
						c.Write(c.Registers.GetHL(), val)
						c.cpuCycles(1)
					}
				}
			// SET 2 SET 3
			case 0xD:
				if secondNibble < 0x8 {
					// SET 2
					if secondNibble != 0x6 {
						reg, val := c.CBLookUp(uint8(secondNibble))
						c.Registers.SetReg(reg, c.SetBit(val, 2, true))
					} else {
						val := c.Read(c.Registers.GetHL())
						c.cpuCycles(1)
						val = c.SetBit(val, 2, true)
						c.Write(c.Registers.GetHL(), val)
						c.cpuCycles(1)
					}
				} else {
					// SET 3
					if secondNibble != 0xE {
						reg, val := c.CBLookUp(uint8(secondNibble))
						c.Registers.SetReg(reg, c.SetBit(val, 3, true))
					} else {
						val := c.Read(c.Registers.GetHL())
						c.cpuCycles(1)
						val = c.SetBit(val, 3, true)
						c.Write(c.Registers.GetHL(), val)
						c.cpuCycles(1)
					}
				}
			// SET 4 SET 5
			case 0xE:
				if secondNibble < 0x8 {
					// SET 4
					if secondNibble != 0x6 {
						reg, val := c.CBLookUp(uint8(secondNibble))
						c.Registers.SetReg(reg, c.SetBit(val, 4, true))
					} else {
						val := c.Read(c.Registers.GetHL())
						c.cpuCycles(1)
						val = c.SetBit(val, 4, true)
						c.Write(c.Registers.GetHL(), val)
						c.cpuCycles(1)
					}
				} else {
					// SET 5
					if secondNibble != 0xE {
						reg, val := c.CBLookUp(uint8(secondNibble))
						c.Registers.SetReg(reg, c.SetBit(val, 5, true))
					} else {
						val := c.Read(c.Registers.GetHL())
						c.cpuCycles(1)
						val = c.SetBit(val, 5, true)
						c.Write(c.Registers.GetHL(), val)
						c.cpuCycles(1)
					}
				}
			// SET 6 SET 7
			case 0xF:
				if secondNibble < 0x8 {
					// SET 6
					if secondNibble != 0x6 {
						reg, val := c.CBLookUp(uint8(secondNibble))
						c.Registers.SetReg(reg, c.SetBit(val, 6, true))
					} else {
						val := c.Read(c.Registers.GetHL())
						c.cpuCycles(1)
						val = c.SetBit(val, 6, true)
						c.Write(c.Registers.GetHL(), val)
						c.cpuCycles(1)
					}
				} else {
					// SET 7
					if secondNibble != 0xE {
						reg, val := c.CBLookUp(uint8(secondNibble))
						c.Registers.SetReg(reg, c.SetBit(val, 7, true))
					} else {
						val := c.Read(c.Registers.GetHL())
						c.cpuCycles(1)
						val = c.SetBit(val, 7, true)
						c.Write(c.Registers.GetHL(), val)
						c.cpuCycles(1)
					}
				}
			}
		},
	},
	0xCC: {
		Mnemonic: "CALL_Z_A16",
		Size:     3,
		Ticks:    []uint8{24, 12},
		AddrMode: R_A16,
		Operation: func(c *CPU) {
			if c.Registers.GetFlag(registers.ZERO_FLAG) {
				c.StackPush16(c.Registers.GetPC())
				c.Registers.SetPC(c.Fetched)
				c.cpuCycles(1)
			}
		},
	},
	0xCD: {
		Mnemonic: "CALL_A16",
		Size:     3,
		Ticks:    []uint8{24},
		AddrMode: A16_R,
		Operation: func(c *CPU) {
			c.StackPush16(c.Registers.GetPC())
			c.Registers.SetPC(c.Fetched)
			c.cpuCycles(1)
		},
	},
	0xCE: {
		Mnemonic: "ADC_A_N8",
		Size:     2,
		Ticks:    []uint8{8},
		AddrMode: R_N8,
		Operation: func(c *CPU) {
			a := c.Registers.GetReg(registers.A)
			carryFlag := c.SetAdcFlags(a, uint8(c.Fetched))
			c.Registers.SetReg(registers.A, a+(uint8(c.Fetched)+carryFlag))
		},
	},
	0xCF: {
		Mnemonic: "RST_$08",
		Size:     3,
		Ticks:    []uint8{16},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			c.StackPush16(c.Registers.GetPC())
			c.Registers.SetPC(0x0008)
			c.cpuCycles(1)
		},
	},
	0xD0: {
		Mnemonic: "RET_NC",
		Size:     1,
		Ticks:    []uint8{20, 8},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			if !c.Registers.GetFlag(registers.CARRY_FLAG) {
				val := c.StackPop16()
				c.Registers.SetPC(val)
				c.cpuCycles(1)
			}
			c.cpuCycles(1)
		},
	},
	0xD1: {
		Mnemonic: "POP_DE",
		Size:     1,
		Ticks:    []uint8{12},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			val := c.StackPop16()
			c.Registers.SetDE(val)
			c.cpuCycles(1)
		},
	},
	0xD2: {
		Mnemonic: "JP_NC_A16",
		Size:     3,
		Ticks:    []uint8{16, 12},
		AddrMode: R_A16,
		Operation: func(c *CPU) {
			if !c.Registers.GetFlag(registers.CARRY_FLAG) {
				c.Registers.SetPC(c.Fetched)
				c.cpuCycles(1)
			}
		},
	},
	0xD3: DASH,
	0xD4: {
		Mnemonic: "CALL_NC_A16",
		Size:     3,
		Ticks:    []uint8{24, 12},
		AddrMode: R_A16,
		Operation: func(c *CPU) {
			if !c.Registers.GetFlag(registers.CARRY_FLAG) {
				c.StackPush16(c.Registers.GetPC())
				c.Registers.SetPC(c.Fetched)
				c.cpuCycles(1)
			}
		},
	},
	0xD5: {
		Mnemonic: "PUSH_DE",
		Size:     1,
		Ticks:    []uint8{16},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			c.StackPush16(c.Registers.GetDE())
			c.cpuCycles(1)
		},
	},
	0xD6: {
		Mnemonic: "SUB_A_N8",
		Size:     1,
		Ticks:    []uint8{8},
		AddrMode: R_N8,
		Operation: func(c *CPU) {
			a := c.Registers.GetReg(registers.A)
			c.SetSubFlags(a, uint8(c.Fetched))
			c.Registers.SetReg(registers.A, a-uint8(c.Fetched))
		},
	},
	0xD7: {
		Mnemonic: "RST_$10",
		Size:     3,
		Ticks:    []uint8{16},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			c.StackPush16(c.Registers.GetPC())
			c.Registers.SetPC(0x0010)
			c.cpuCycles(1)
		},
	},
	0xD8: {
		Mnemonic: "RET_C",
		Size:     1,
		Ticks:    []uint8{20, 8},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			if c.Registers.GetFlag(registers.CARRY_FLAG) {
				val := c.StackPop16()
				c.Registers.SetPC(val)
				c.cpuCycles(1)
			}
			c.cpuCycles(1)
		},
	},
	0xD9: {
		Mnemonic: "RETI",
		Size:     1,
		Ticks:    []uint8{16},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			val := c.StackPop16()
			c.Registers.SetPC(val)
			c.cpuCycles(1)
			c.Registers.SetIME(true)
		},
	},
	0xDA: {
		Mnemonic: "JP_C_A16",
		Size:     3,
		Ticks:    []uint8{16, 12},
		AddrMode: R_A16,
		Operation: func(c *CPU) {
			if c.Registers.GetFlag(registers.CARRY_FLAG) {
				c.Registers.SetPC(c.Fetched)
				c.cpuCycles(1)
			}
			c.cpuCycles(1)
		},
	},
	0xDB: DASH,
	0xDC: {
		Mnemonic: "CALL_C_A16",
		Size:     3,
		Ticks:    []uint8{24, 12},
		AddrMode: R_A16,
		Operation: func(c *CPU) {
			if c.Registers.GetFlag(registers.CARRY_FLAG) {
				c.StackPush16(c.Registers.GetPC())
				c.Registers.SetPC(c.Fetched)
				c.cpuCycles(1)
			}
		},
	},
	0xDD: DASH,
	0xDE: {
		Mnemonic: "SBC_A_N8",
		Size:     1,
		Ticks:    []uint8{8},
		AddrMode: R_N8,
		Operation: func(c *CPU) {
			a := c.Registers.GetReg(registers.A)
			carryFlag := c.SetSbcFlags(a, uint8(c.Fetched))
			c.Registers.SetReg(registers.A, a-(uint8(c.Fetched)-carryFlag))
		},
	},
	0xDF: {
		Mnemonic: "RST_$18",
		Size:     3,
		Ticks:    []uint8{16},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			c.StackPush16(c.Registers.GetPC())
			c.Registers.SetPC(0x0018)
			c.cpuCycles(1)
		},
	},
	0xE0: {
		Mnemonic: "LDH_[A8]_A",
		Size:     2,
		AddrMode: A8_A,
		Ticks:    []uint8{12},
		Operation: func(c *CPU) {
			c.Write(c.Fetched, c.Registers.GetReg(registers.A))
			c.cpuCycles(1)
		},
	},
	0xE1: {
		Mnemonic: "POP_HL",
		Size:     1,
		Ticks:    []uint8{12},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			val := c.StackPop16()
			c.Registers.SetHL(val)
			c.cpuCycles(1)
		},
	},
	0xE2: {
		Mnemonic: "LDH_[C]_A",
		Size:     1,
		AddrMode: NONE,
		Ticks:    []uint8{8},
		Operation: func(c *CPU) {
			c.Write(uint16(c.Registers.GetReg(registers.C))+0xFF00, c.Registers.GetReg(registers.A))
			c.cpuCycles(1)
		},
	},
	0xE3: DASH,
	0xE4: DASH,
	0xE5: {
		Mnemonic: "PUSH_HL",
		Size:     1,
		Ticks:    []uint8{16},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			c.StackPush16(c.Registers.GetHL())
			c.cpuCycles(1)
		},
	},
	0xE6: {
		Mnemonic: "AND_A_N8",
		Size:     1,
		Ticks:    []uint8{8},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			a := c.Registers.GetReg(registers.A)
			c.Registers.SetReg(registers.A, a&uint8(c.Fetched))
			c.SetAndFlags(c.Registers.GetReg(registers.A))
		},
	},
	0xE7: {
		Mnemonic: "RST_$20",
		Size:     3,
		Ticks:    []uint8{16},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			c.StackPush16(c.Registers.GetPC())
			c.Registers.SetPC(0x0020)
			c.cpuCycles(1)
		},
	},
	0xE8: {
		Mnemonic: "ADD_SP_E8",
		Size:     2,
		Ticks:    []uint8{16},
		AddrMode: E8,
		Operation: func(c *CPU) {
			sp := c.Registers.GetSP()
			c.SetAddFlags16(sp, uint16(c.RelAddr))
			c.Registers.SetFlag(registers.ZERO_FLAG, false)
			c.cpuCycles(1)
			c.Registers.SetSP(sp + uint16(c.RelAddr))
			c.cpuCycles(1)
		},
	},
	0xE9: {
		Mnemonic: "JP_HL",
		Size:     3,
		Ticks:    []uint8{4},
		AddrMode: R_A16,
		Operation: func(c *CPU) {
			c.Registers.SetPC(c.Registers.GetHL())
		},
	},
	0xEA: {
		Mnemonic: "LD_[A16]_A",
		Size:     3,
		AddrMode: A16_R,
		Operation: func(c *CPU) {
			c.Write(c.Fetched, c.Registers.GetReg(registers.A))
			c.cpuCycles(1)
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
		Operation: func(c *CPU) {
			a := c.Registers.GetReg(registers.A)
			c.Registers.SetReg(registers.A, a^uint8(c.Fetched))
			c.SetXorFlags(c.Registers.GetReg(registers.A))
		},
	},
	0xEF: {
		Mnemonic: "RST_$EF",
		Size:     3,
		Ticks:    []uint8{16},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			c.StackPush16(c.Registers.GetPC())
			c.Registers.SetPC(0x00EF)
			c.cpuCycles(1)
		},
	},
	0xF0: {
		Mnemonic: "LDH_A_[A8]",
		Size:     2,
		Ticks:    []uint8{12},
		AddrMode: A_A8,
		Operation: func(c *CPU) {
			c.Registers.SetReg(registers.A, c.Read(c.Fetched))
			c.cpuCycles(1)
		},
	},
	0xF1: {
		Mnemonic: "POP_AF",
		Size:     1,
		Ticks:    []uint8{12},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			val := c.StackPop16()
			c.Registers.SetAF(val)
			c.cpuCycles(1)
		},
	},
	0xF2: {
		Mnemonic: "LDH_A_[C]",
		Size:     1,
		AddrMode: NONE,
		Ticks:    []uint8{8},
		Operation: func(c *CPU) {
			val := c.Read(uint16(c.Registers.GetReg(registers.C)) + 0xFF00)
			c.Registers.SetReg(registers.A, val)
			c.cpuCycles(1)
		},
	},
	0xF3: {
		Mnemonic: "DI",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			c.Registers.SetIME(false)
		},
	},
	0xF4: DASH,
	0xF5: {
		Mnemonic: "PUSH_AF",
		Size:     1,
		Ticks:    []uint8{16},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			c.StackPush16(c.Registers.GetAF())
			c.cpuCycles(1)
		},
	},
	0xF6: {
		Mnemonic: "OR_A_N8",
		Size:     1,
		Ticks:    []uint8{8},
		AddrMode: R_N8,
		Operation: func(c *CPU) {
			a := c.Registers.GetReg(registers.A)
			c.Registers.SetReg(registers.A, a|uint8(c.Fetched))
			c.SetXorFlags(c.Registers.GetReg(registers.A))
		},
	},
	0xF7: {
		Mnemonic: "RST_$30",
		Size:     3,
		Ticks:    []uint8{16},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			c.StackPush16(c.Registers.GetPC())
			c.Registers.SetPC(0x0030)
			c.cpuCycles(1)
		},
	},
	0xF8: {
		Mnemonic: "LD_HL_SP+E8",
		Size:     2,
		Ticks:    []uint8{12},
		AddrMode: E8,
		Operation: func(c *CPU) {
			c.Registers.SetHL(c.Registers.GetSP() + uint16(c.RelAddr))
			c.cpuCycles(1)
		},
	},
	0xF9: {
		Mnemonic: "LD_SP_HL",
		Size:     1,
		Ticks:    []uint8{8},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			c.Registers.SetSP(c.Registers.GetHL())
		},
	},
	0xFA: {
		Mnemonic: "LD_A_[A16]",
		Size:     3,
		Ticks:    []uint8{16},
		AddrMode: A16_R,
		Operation: func(c *CPU) {
			c.Registers.SetReg(registers.A, c.Read(c.Fetched))
			c.cpuCycles(1)
		},
	},
	0xFB: {
		Mnemonic: "EI",
		Size:     1,
		Ticks:    []uint8{4},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			c.EnablingIME = true
		},
	},
	0xFC: DASH,
	0xFD: DASH,
	0xFE: {
		Mnemonic: "CP_A_N8",
		Size:     1,
		Ticks:    []uint8{8},
		AddrMode: R_N8,
		Operation: func(c *CPU) {
			c.SetCpFlags(c.Registers.GetReg(registers.A), uint8(c.Fetched))
		},
	},
	0xFF: {
		Mnemonic: "RST_$38",
		Size:     3,
		Ticks:    []uint8{16},
		AddrMode: NONE,
		Operation: func(c *CPU) {
			c.StackPush16(c.Registers.GetPC())
			c.Registers.SetPC(0x0038)
			c.cpuCycles(1)
		},
	},
}

var DASH = Instruction{
	Mnemonic: "-",
	Size:     1,
	AddrMode: NONE,
	Operation: func(c *CPU) {
	},
}

func (c *CPU) CBLookUp(highNibble uint8) (registers.Reg, uint8) {
	switch highNibble {
	case 0x0, 0x8:
		return registers.B, c.Registers.GetReg(registers.B)
	case 0x1, 0x9:
		return registers.C, c.Registers.GetReg(registers.C)
	case 0x2, 0xA:
		return registers.D, c.Registers.GetReg(registers.D)
	case 0x3, 0xB:
		return registers.E, c.Registers.GetReg(registers.E)
	case 0x4, 0xC:
		return registers.H, c.Registers.GetReg(registers.H)
	case 0x5, 0xD:
		return registers.L, c.Registers.GetReg(registers.L)
	default:
		return registers.A, c.Registers.GetReg(registers.A)
	}
}
