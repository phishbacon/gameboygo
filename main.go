package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"

	"github.com/jupiterrider/purego-sdl3/sdl"
	"github.com/phishbacon/gameboygo/cart"
	"github.com/phishbacon/gameboygo/display"
	"github.com/phishbacon/gameboygo/soc"
	"github.com/phishbacon/gameboygo/util"
)

type Gameboy struct {
	soc     *soc.SOC
	cart    *cart.Cart
	running bool
	paused  bool
	display *display.Display
}

func NewGoboy(args []string) *Gameboy {

	goboy := &Gameboy{}
	soc := soc.NewSOC()
	display := new(display.Display)
	goboy.soc = soc
	goboy.display = display
	goboy.running = false

	return goboy
}

func (g *Gameboy) Start() {
	g.running = true
	g.display.Start()
	go g.soc.Init()
}

func (g *Gameboy) LoadCart(fileName string) {
	dump, dumpErr := os.ReadFile(fileName)
	if dumpErr != nil {
		fmt.Print(dumpErr)
		os.Exit(-1)
	}
	// give bus reference to cart so soc components can read from it
	g.cart = (*cart.Cart)(&dump)
	g.soc.ConnectCart(&dump)

	var cartHeader cart.CartHeader
	headerErr := binary.Read(bytes.NewReader((*g.cart)[0x0100:0x0150]), binary.LittleEndian, &cartHeader)
	if headerErr != nil {
		fmt.Print(headerErr)
		os.Exit(-1)
	}

	fmt.Printf("\nTITLE      %s\n", string(cartHeader.Title[:]))
	fmt.Printf("LIC        %s\n", cartHeader.GetCartLicName())
	fmt.Printf("SGB        %x\n", cartHeader.SGBFlag)
	fmt.Printf("TYPE       %s\n", cartHeader.GetCartTypeName())
	fmt.Printf("ROM SIZE   %d KB\n", 32<<cartHeader.ROMSize)
	fmt.Printf("RAM SIZE   %s\n", cartHeader.GetRAMSize())
	fmt.Printf("DEST CODE  %s\n", cartHeader.GetDestCode())
	fmt.Printf("VERSION    %d\n", cartHeader.Version)

	var checksum uint8 = 0
	for address := 0x0134; address <= 0x014C; address++ {
		checksum = checksum - (*g.cart)[address] - 1
	}

	checksumPassed := util.If(cartHeader.HeaderChecksum == (checksum&0x00FF), "PASSED", "FAILED")
	fmt.Printf("CHECKSUM   %s\n", checksumPassed)
	g.cart.DumpHex()
}

func main() {

	args := os.Args

	gameboy := NewGoboy(args)
	if len(args) < 2 {
		fmt.Println("Rom required")
		os.Exit(-1)
	}

	gameboy.LoadCart(args[1])

	fmt.Printf("Loading %s\n", args[1])
	if gameboy.cart.VerifyLogoDump() {
		// jump to address 0x0100
	} else {
		fmt.Println("Failed to verify logo")
		os.Exit(-1)
	}

	gameboy.Start()
	sdl.Init(sdl.InitVideo)
	defer sdl.Quit()
	defer sdl.DestroyRenderer(gameboy.display.ScreenRenderer)
	defer sdl.DestroyWindow(gameboy.display.Screen)
	for gameboy.display.On {
		gameboy.soc.Paused = gameboy.display.Paused
		gameboy.display.EventLoop()
		gameboy.display.Render()
	}
}
