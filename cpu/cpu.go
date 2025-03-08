package cpu

import (
	"fmt"
	"os"

	"github.com/phishbacon/gameboygo/io"
)

const IF uint16 = 0xFF0F
const IE uint16 = 0xFFFF

type CPU struct {
	Registers *Registers
	CurInst   *Instruction

	Fetched        uint16
	Paused         bool
	DestAddr       uint16
	RelAddr        int8
	Ticks          uint64
	PrevTicks      uint64
	Halted         bool
	EnablingIME    bool
	Read           func(uint16) uint8
	Write          func(uint16, uint8)
	CPUStateString string
}

func NewCPU() *CPU {
	registers := new(Registers)
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
	c.Registers.SP = 0xFFFE
	c.Registers.PC = 0x0100
}

func (c *CPU) StackPush(value uint8) {
	c.Registers.SP--
	c.Write(c.Registers.SP, value)
	c.cpuCycles(1)
}

func (c *CPU) StackPush16(value uint16) {
	// push hi
	c.StackPush(uint8((value & 0xFF00) >> 8))
	// push lo
	c.StackPush(uint8(value & 0x00FF))
}

func (c *CPU) StackPop() uint8 {
	poppedValue := c.Read(c.Registers.SP)
	c.Registers.SP++
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

func (c *CPU) execute() string {
	pc := c.Registers.PC
	opcode := c.Read(pc)
	c.PrevTicks = c.Ticks
	c.cpuCycles(1)
	if Instructions[opcode].Operation == nil {
		fmt.Printf("opcode: %04x not implemented\n", opcode)
		fmt.Printf("%02x 02%d 02%d\n", opcode, c.Read(pc+1), c.Read(pc+2))
		os.Exit(-1)
	}

	return c.process(opcode)
}

func (c *CPU) process(opcode uint8) string {
	c.CPUStateString = ""
	c.CurInst = &Instructions[opcode]
	pc := c.Registers.PC
	pc1 := c.Read(pc + 1)
	pc2 := c.Read(pc + 2)
	c.CPUStateString += fmt.Sprintf("%-10s %02x %02x %02x\t",
		c.CurInst.Mnemonic,
		opcode,
		pc1,
		pc2)
	c.Registers.PC++
	c.CurInst.AddrMode(c)
	ticksIndex := c.CurInst.Operation(c)
	var z, n, h, carry string
	if c.Registers.GetFlag(ZERO_FLAG) {
		z = "Z"
	} else {
		z = "-"
	}
	if c.Registers.GetFlag(SUBTRACTION_FLAG) {
		n = "N"
	} else {
		n = "-"
	}
	if c.Registers.GetFlag(HALF_CARRY_FLAG) {
		h = "H"
	} else {
		h = "-"
	}
	if c.Registers.GetFlag(CARRY_FLAG) {
		carry = "C"
	} else {
		carry = "-"
	}
	c.CPUStateString += fmt.Sprintf("A: 0x%02x F: %s%s%s%s BC: 0x%04x DE: 0x%04x HL: 0x%04x PC: 0x%04x SP: 0x%04x SB: 0x%04x SC: 0x%04x\n",
		c.Registers.A,
		z,
		n,
		h,
		carry,
		c.Registers.GetBC(),
		c.Registers.GetDE(),
		c.Registers.GetHL(),
		c.Registers.PC,
		c.Registers.SP,
		c.Read(0xFF01),
		c.Read(0xFF02))
	// c.Ticks)
	fmt.Print(c.CPUStateString)
	if c.PrevTicks+uint64(c.CurInst.Ticks[ticksIndex]) != c.Ticks {
		fmt.Printf("Ticks before operation: %d, Ticks after %d, Should be %d, exiting", c.PrevTicks, c.Ticks, c.PrevTicks+uint64(c.CurInst.Ticks[ticksIndex]))
		os.Exit(-1)
	}
	return c.CPUStateString
}

func (c *CPU) Step() string {
	cpuString := ""
	if !c.Halted {
		cpuString = c.execute()
	} else {
		c.cpuCycles(1)
		if c.Read(IF)&c.Read(IE) > 0 {
			c.Halted = false
		}
	}

	if c.Registers.IME {
		c.HandleInterupts()
		c.EnablingIME = false
	}

	if c.EnablingIME {
		c.Registers.IME = true
	}
	return cpuString
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
		c.Registers.IME = false
		return true
	}

	return false
}

func (c *CPU) CallInterupt(address uint16, interupt uint8) {
	c.StackPush16(c.Registers.PC)
	c.Registers.PC = address
}
