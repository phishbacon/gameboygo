package cpu

import (
	"fmt"
	"os"

	"github.com/phishbacon/gameboygo/bus"
	"github.com/phishbacon/gameboygo/common"
	"github.com/phishbacon/gameboygo/io"
)

const IF uint16 = 0xFF0F
const IE uint16 = 0xFFFF

type CPU struct {
	Registers    *Registers
	CurInst      *Instruction
	NextByte     uint8
	NextNextByte uint8
	CurOpCode    uint8
	bus          *bus.Bus

	Fetched        uint16
	Paused         bool
	DestAddr       uint16
	RelAddr        int8
	Ticks          uint64
	PrevTicks      uint64
	Halted         bool
	EnablingIME    bool
	CpuStateString string
}

func NewCPU(bus *bus.Bus) *CPU {
	registers := NewRegisters()
	return &CPU{
		Registers: registers,
		bus:       bus,
	}
}

func (c *CPU) Init() {
	c.Registers.AF.Equals(0x01B0)
	c.Registers.BC.Equals(0x0000)
	c.Registers.DE.Equals(0xFF56)
	c.Registers.HL.Equals(0x000D)
	c.Registers.SP.Equals(0xFFFE)
	c.Registers.PC.Equals(0x0100)
}

func (c *CPU) StackPush(value uint8) {
	c.Registers.SP.Sub(1)
	c.bus.Write(c.Registers.SP.Value(), value)
	c.cpuCycles(1)
}

func (c *CPU) StackPush16(value uint16) {
	// push hi
	c.StackPush(uint8((value & 0xFF00) >> 8))
	// push lo
	c.StackPush(uint8(value & 0x00FF))
}

func (c *CPU) StackPop() uint8 {
	poppedValue := c.bus.Read(c.Registers.SP.Value())
	c.Registers.SP.Add(1)
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
	pc := c.Registers.PC.Value()
	opcode := c.bus.Read(pc)
	c.PrevTicks = c.Ticks
	c.cpuCycles(1)
	if Instructions[opcode].Operation == nil {
		fmt.Printf("opcode: %04x not implemented\n", opcode)
		fmt.Printf("%02x 02%d 02%d\n", opcode, c.bus.Read(pc+1), c.bus.Read(pc+2))
		os.Exit(-1)
	}

	c.process(opcode)
}

func (c *CPU) process(opcode uint8) {
	c.CpuStateString = ""
	c.CurOpCode = opcode
	c.CurInst = &Instructions[opcode]
	pc := c.Registers.PC.Value()
	c.NextByte = c.bus.Read(pc + 1)
	c.NextNextByte = c.bus.Read(pc + 2)
	pc1 := c.NextByte
	pc2 := c.NextNextByte
	c.CpuStateString += fmt.Sprintf("%-10s %02x %02x %02x\t",
		c.CurInst.Mnemonic,
		opcode,
		pc1,
		pc2)
	c.Registers.PC.Add(1)
	c.CurInst.AddrMode(c)
	ticksIndex := c.CurInst.Operation(c)
	var z, n, h, carry string
	if c.Registers.GetFlag(common.ZERO_FLAG) {
		z = "Z"
	} else {
		z = "-"
	}
	if c.Registers.GetFlag(common.SUBTRACTION_FLAG) {
		n = "N"
	} else {
		n = "-"
	}
	if c.Registers.GetFlag(common.HALF_CARRY_FLAG) {
		h = "H"
	} else {
		h = "-"
	}
	if c.Registers.GetFlag(common.CARRY_FLAG) {
		carry = "C"
	} else {
		carry = "-"
	}
	c.CpuStateString += fmt.Sprintf("\nA: 0x%02x\nF: %s%s%s%s\nBC: 0x%04x\nDE: 0x%04x\nHL: 0x%04x\nPC: 0x%04x\nSP: 0x%04x\nSB: 0x%04x\nSC: 0x%04x\n",
		c.Registers.A.Value(),
		z,
		n,
		h,
		carry,
		c.Registers.BC.Value(),
		c.Registers.DE.Value(),
		c.Registers.HL.Value(),
		c.Registers.PC.Value(),
		c.Registers.SP.Value(),
		c.bus.Read(0xFF01),
		c.bus.Read(0xFF02))
	// c.Ticks)
	fmt.Print(c.CpuStateString)
	if c.PrevTicks+uint64(ticks[opcode][ticksIndex]) != c.Ticks {
		fmt.Printf("Ticks before operation: %d, Ticks after %d, Should be %d, exiting", c.PrevTicks, c.Ticks, c.PrevTicks+uint64(ticks[opcode][ticksIndex]))
		os.Exit(-1)
	}
}

func (c *CPU) Step() {
	if !c.Halted {
		c.execute()
	} else {
		c.cpuCycles(1)
		if c.bus.Read(IF)&c.bus.Read(IE) > 0 {
			c.Halted = false
		}
	}

	if c.Registers.IME.Value() == 1 {
		c.HandleInterupts()
		c.EnablingIME = false
	}

	if c.EnablingIME {
		c.Registers.IME.Equals(1)
	}
}

func (c *CPU) HandleInterupts() {
	interuptsFlag := c.bus.Read(IF)
	interuptsEnabled := c.bus.Read(IE)
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
		c.bus.Write(IF, inf & ^interupt)
		c.Halted = false
		c.Registers.IME.Equals(0)
		return true
	}

	return false
}

func (c *CPU) CallInterupt(address uint16, interupt uint8) {
	c.StackPush16(c.Registers.PC.Value())
	c.Registers.PC.Equals(address)
}
