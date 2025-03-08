package display

import (
	"fmt"

	"github.com/jupiterrider/purego-sdl3/sdl"
)

type Display struct {
	Screen              *sdl.Window
	ScreenRenderer      *sdl.Renderer
	On                  bool
	Paused              bool
}

func (d *Display) Start() {
	d.Paused = true
	if !sdl.CreateWindowAndRenderer("gameboygo", 160, 144, 0, &d.Screen, &d.ScreenRenderer) {
		panic(sdl.GetError())
	}

	sdl.SetRenderDrawColor(d.ScreenRenderer, 100, 150, 200, 255)

	d.On = true
}

func (d *Display) EventLoop() {
	var event sdl.Event
	for sdl.PollEvent(&event) {
		switch event.Type() {
		case sdl.EventQuit:
				d.On = false
		case sdl.EventKeyDown:
			if event.Key().Scancode == sdl.ScancodeSpace {
				fmt.Println(d.Paused)
				d.Paused = !d.Paused
			}
		}
	}
}

func (d *Display) Render() {
	sdl.RenderClear(d.ScreenRenderer)
	sdl.RenderPresent(d.ScreenRenderer)
}
