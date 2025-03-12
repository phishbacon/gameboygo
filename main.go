package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"

	"github.com/jupiterrider/purego-sdl3/sdl"
	"github.com/phishbacon/gameboygo/cart"
	"github.com/phishbacon/gameboygo/common"
	"github.com/phishbacon/gameboygo/soc"
)

type Gameboy struct {
	SOC     *soc.SOC
	Cart    *cart.Cart
	Running bool
	Paused  bool
	Display *Display
}

func NewGoboy(args []string) *Gameboy {

	goboy := &Gameboy{}
	soc := soc.NewSOC()
	goboy.SOC = soc
	goboy.Running = false

	return goboy
}

func (g *Gameboy) Start() {
	g.Running = true
	g.Paused = true
	g.InitDisplay()
	go g.SOC.Init()
}

func (g *Gameboy) LoadCart(fileName string) {
	dump, dumpErr := os.ReadFile(fileName)
	if dumpErr != nil {
		fmt.Print(dumpErr)
		os.Exit(-1)
	}
	// give bus reference to cart so soc components can read from it
	g.Cart = (*cart.Cart)(&dump)
	g.SOC.ConnectCart(&dump)

	var cartHeader cart.CartHeader
	headerErr := binary.Read(bytes.NewReader((*g.Cart)[0x0100:0x0150]), binary.LittleEndian, &cartHeader)
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
		checksum = checksum - (*g.Cart)[address] - 1
	}

	checksumPassed := common.If(cartHeader.HeaderChecksum == (checksum&0x00FF), "PASSED", "FAILED")
	fmt.Printf("CHECKSUM   %s\n", checksumPassed)
	g.Cart.DumpHex()
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
	if gameboy.Cart.VerifyLogoDump() {
		// jump to address 0x0100
	} else {
		fmt.Println("Failed to verify logo")
		os.Exit(-1)
	}

	gameboy.Start()
	sdl.Init(sdl.InitVideo)
	defer sdl.Quit()
	defer sdl.DestroyRenderer(gameboy.Display.ScreenRenderer)
	defer sdl.DestroyWindow(gameboy.Display.Screen)
	defer sdl.DestroyRenderer(gameboy.Display.DebugScreenRenderer)
	defer sdl.DestroyWindow(gameboy.Display.DebugScreen)
	for gameboy.Running {
		gameboy.EventLoop()
		gameboy.DebugScreenRender()
		gameboy.ScreenRender()
	}
}
