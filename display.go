package main

import (
	"github.com/jupiterrider/purego-sdl3/sdl"
	"github.com/phishbacon/gameboygo/common"
)

var DebugWidth int32 = 1000
var DebugHeight int32 = 1000

type Display struct {
	Screen              *sdl.Window
	ScreenRenderer      *sdl.Renderer
	DebugScreen         *sdl.Window
	DebugScreenRenderer *sdl.Renderer
	TextOffsetY         float32
	DebugStepValue      uint8
}

func (g *Gameboy) InitDisplay() {
	g.Display = new(Display)
	g.Display.DebugStepValue = 200
	if !sdl.CreateWindowAndRenderer("gameboygo", 160, 144, 0, &g.Display.Screen, &g.Display.ScreenRenderer) {
		panic(sdl.GetError())
	}

	if !sdl.CreateWindowAndRenderer("debug gameboygo", DebugWidth, DebugHeight, sdl.WindowResizable, &g.Display.DebugScreen, &g.Display.DebugScreenRenderer) {
		panic(sdl.GetError())
	}

	sdl.SetRenderDrawColor(g.Display.ScreenRenderer, 100, 150, 200, 255)
}

func (g *Gameboy) EventLoop() {
	var event sdl.Event
	for sdl.PollEvent(&event) {
		switch event.Type() {
		case sdl.EventWindowCloseRequested:
			g.Running = false
		case sdl.EventKeyDown:
			if event.Key().Scancode == sdl.ScancodeSpace {
				g.SOC.Paused = !g.SOC.Paused
			} else if event.Key().Scancode == sdl.ScancodeS && g.Paused {
				g.SOC.Step(g.Display.DebugStepValue)
			} else if event.Key().Scancode == sdl.ScancodeDown {
				g.Display.DebugStepValue--
			} else if event.Key().Scancode == sdl.ScancodeUp {
				g.Display.DebugStepValue++
			}
		case sdl.EventMouseWheel:
			g.Display.TextOffsetY += event.Wheel().Y * 10
			if g.Display.TextOffsetY < 0 {
				g.Display.TextOffsetY = 0
			}
		}
	}
}

func (g *Gameboy) ScreenRender() {
	sdl.RenderClear(g.Display.ScreenRenderer)
	sdl.RenderPresent(g.Display.ScreenRenderer)
}

func (g *Gameboy) DebugScreenRender() {
	offset := g.Display.TextOffsetY
	memory := g.SOC.Bus.DumpMemory()
	cpu := g.SOC.CPU
	pc1 := cpu.NextByte
	pc2 := cpu.NextNextByte
	curInst := cpu.CurInst
	curOpCode := cpu.CurOpCode
	if curInst == nil {
		return
	}
	var z, n, h, c string
	if cpu.Registers.GetFlag(common.ZERO_FLAG) {
		z = "Z"
	} else {
		z = "-"
	}
	if cpu.Registers.GetFlag(common.SUBTRACTION_FLAG) {
		n = "N"
	} else {
		n = "-"
	}
	if cpu.Registers.GetFlag(common.HALF_CARRY_FLAG) {
		h = "H"
	} else {
		h = "-"
	}
	if cpu.Registers.GetFlag(common.CARRY_FLAG) {
		c = "C"
	} else {
		c = "-"
	}
	sdl.SetRenderDrawColor(g.Display.DebugScreenRenderer, 0, 0, 0, sdl.AlphaOpaque) /* black, full alpha */
	sdl.RenderClear(g.Display.DebugScreenRenderer)                                  /* start with a blank canvas. */
	sdl.SetRenderDrawColor(g.Display.DebugScreenRenderer, 255, 255, 255, sdl.AlphaOpaque)
	var scale float32 = 3.0
	sdl.SetRenderScale(g.Display.DebugScreenRenderer, scale, scale)
	// current cpu instruction
	sdl.RenderDebugTextFormat(g.Display.DebugScreenRenderer, 175, 1, "%-10s %02x %02x %02x",
		curInst.Mnemonic,
		curOpCode,
		pc1,
		pc2,
	)
	sdl.RenderDebugTextFormat(g.Display.DebugScreenRenderer, 150, 1, "%d",
		g.Display.DebugStepValue,
	)
	sdl.RenderDebugTextFormat(g.Display.DebugScreenRenderer, 150, 91, "%d",
		g.SOC.TotalSteps,
	)
	sdl.RenderDebugTextFormat(g.Display.DebugScreenRenderer, 175, 11, " A: 0x%02x",
		cpu.Registers.A,
	)
	sdl.RenderDebugTextFormat(g.Display.DebugScreenRenderer, 175, 21, " F: %s%s%s%s", z, n, h, c)
	sdl.RenderDebugTextFormat(g.Display.DebugScreenRenderer, 175, 31, " F: %08b", cpu.Registers.F)
	sdl.RenderDebugTextFormat(g.Display.DebugScreenRenderer, 175, 41, "BC: 0x%04x", cpu.Registers.GetBC())
	sdl.RenderDebugTextFormat(g.Display.DebugScreenRenderer, 175, 51, "DE: 0x%04x", cpu.Registers.GetDE())
	sdl.RenderDebugTextFormat(g.Display.DebugScreenRenderer, 175, 61, "HL: 0x%04x", cpu.Registers.GetHL())
	sdl.RenderDebugTextFormat(g.Display.DebugScreenRenderer, 175, 71, "PC: 0x%04x", cpu.Registers.PC)
	sdl.RenderDebugTextFormat(g.Display.DebugScreenRenderer, 175, 81, "SP: 0x%04x", cpu.Registers.SP)
	sdl.SetRenderScale(g.Display.DebugScreenRenderer, scale/3, scale/3)
	var y float32 = 1
	for i := 0; i < len(memory)-16; i += 16 {
		sdl.RenderDebugTextFormat(g.Display.DebugScreenRenderer, 0, y-(offset),
			"0x%04x: %02x %02x %02x %02x %02x %02x %02x %02x %02x %02x %02x %02x %02x %02x %02x %02x ",
			i,
			memory[i], memory[i+1], memory[i+2], memory[i+3],
			memory[i+4], memory[i+5], memory[i+6], memory[i+7],
			memory[i+8], memory[i+9], memory[i+10], memory[i+11],
			memory[i+12], memory[i+13], memory[i+14], memory[i+15],
		)
		y += 10
	}

	sdl.RenderPresent(g.Display.DebugScreenRenderer) /* put it all on the screen! */
}
