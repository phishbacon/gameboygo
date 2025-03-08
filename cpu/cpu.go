package cpu

import (
	"fmt"
	"os"

	"github.com/phishbacon/gameboygo/bus"
	"github.com/phishbacon/gameboygo/io"
)

const IF uint16 = 0xFF0F
const IE uint16 = 0xFFFF

type CPU struct {
	registers *Registers
	curInst   *Instruction
	bus       *bus.Bus

	fetched        uint16
	paused         bool
	destAddr       uint16
	relAddr        int8
	ticks          uint64
	prevTicks      uint64
	halted         bool
	enablingIME    bool
	cpuStateString string
}

func NewCPU(bus *bus.Bus) *CPU {
	registers := new(Registers)
	return &CPU{
		registers: registers,
		bus: bus,
	}
}

func (c *CPU) Init() {
	c.registers.SetAF(0x01B0)
	c.registers.SetBC(0x0000)
	c.registers.SetDE(0xFF56)
	c.registers.SetHL(0x000D)
	c.registers.SP = 0xFFFE
	c.registers.PC = 0x0100
}

func (c *CPU) StackPush(value uint8) {
	c.registers.SP--
	c.bus.Write(c.registers.SP, value)
	c.cpuCycles(1)
}

func (c *CPU) StackPush16(value uint16) {
	// push hi
	c.StackPush(uint8((value & 0xFF00) >> 8))
	// push lo
	c.StackPush(uint8(value & 0x00FF))
}

func (c *CPU) StackPop() uint8 {
	poppedValue := c.bus.Read(c.registers.SP)
	c.registers.SP++
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
	pc := c.registers.PC
	opcode := c.bus.Read(pc)
	c.prevTicks = c.ticks
	c.cpuCycles(1)
	if Instructions[opcode].Operation == nil {
		fmt.Printf("opcode: %04x not implemented\n", opcode)
		fmt.Printf("%02x 02%d 02%d\n", opcode, c.bus.Read(pc+1), c.bus.Read(pc+2))
		os.Exit(-1)
	}

	return c.process(opcode)
}

func (c *CPU) process(opcode uint8) string {
	c.cpuStateString = ""
	c.curInst = &Instructions[opcode]
	pc := c.registers.PC
	pc1 := c.bus.Read(pc + 1)
	pc2 := c.bus.Read(pc + 2)
	c.cpuStateString += fmt.Sprintf("%-10s %02x %02x %02x\t",
		c.curInst.Mnemonic,
		opcode,
		pc1,
		pc2)
	c.registers.PC++
	c.curInst.AddrMode(c)
	ticksIndex := c.curInst.Operation(c)
	var z, n, h, carry string
	if c.registers.GetFlag(ZERO_FLAG) {
		z = "Z"
	} else {
		z = "-"
	}
	if c.registers.GetFlag(SUBTRACTION_FLAG) {
		n = "N"
	} else {
		n = "-"
	}
	if c.registers.GetFlag(HALF_CARRY_FLAG) {
		h = "H"
	} else {
		h = "-"
	}
	if c.registers.GetFlag(CARRY_FLAG) {
		carry = "C"
	} else {
		carry = "-"
	}
	c.cpuStateString += fmt.Sprintf("A: 0x%02x F: %s%s%s%s BC: 0x%04x DE: 0x%04x HL: 0x%04x PC: 0x%04x SP: 0x%04x SB: 0x%04x SC: 0x%04x\n",
		c.registers.A,
		z,
		n,
		h,
		carry,
		c.registers.GetBC(),
		c.registers.GetDE(),
		c.registers.GetHL(),
		c.registers.PC,
		c.registers.SP,
		c.bus.Read(0xFF01),
		c.bus.Read(0xFF02))
	// c.Ticks)
	fmt.Print(c.cpuStateString)
	if c.prevTicks+uint64(c.curInst.Ticks[ticksIndex]) != c.ticks {
		fmt.Printf("Ticks before operation: %d, Ticks after %d, Should be %d, exiting", c.prevTicks, c.ticks, c.prevTicks+uint64(c.curInst.Ticks[ticksIndex]))
		os.Exit(-1)
	}
	return c.cpuStateString
}

func (c *CPU) Step() string {
	cpuString := ""
	if !c.halted {
		cpuString = c.execute()
	} else {
		c.cpuCycles(1)
		if c.bus.Read(IF)&c.bus.Read(IE) > 0 {
			c.halted = false
		}
	}

	if c.registers.IME {
		c.HandleInterupts()
		c.enablingIME = false
	}

	if c.enablingIME {
		c.registers.IME = true
	}
	return cpuString
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
		c.halted = false
		c.registers.IME = false
		return true
	}

	return false
}

func (c *CPU) CallInterupt(address uint16, interupt uint8) {
	c.StackPush16(c.registers.PC)
	c.registers.PC = address
}
