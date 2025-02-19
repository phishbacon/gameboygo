package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"goboy/cart"
	"goboy/soc"
	"os"
)

type goboy struct {
	soc     soc.SOC
	cart    cart.Cart
	paused  bool
	running bool
	ticks   uint64
}

func (g *goboy) Start() {
	g.soc = soc.NewSOC()
	g.running = true
	g.paused = false
	g.ticks = 0
}

func (g *goboy) LoadCart(fileName string) {
	dump, dumpErr := os.ReadFile(fileName)
	if dumpErr != nil {
		fmt.Print(dumpErr)
	}
	g.cart = dump

	var cartHeader cart.CartHeader
	headerErr := binary.Read(bytes.NewReader(g.cart[0x0100:0x0150]), binary.LittleEndian, &cartHeader)
	if headerErr != nil {
		fmt.Print(headerErr)
	}

  fmt.Print(cartHeader)

	fmt.Printf("\nTITLE      %s\n", string(cartHeader.Title[:]))
	fmt.Printf("LIC        %s\n", cartHeader.GetCartLicName())
	fmt.Printf("SGB        %x\n", cartHeader.SGBFlag)
	fmt.Printf("TYPE       %s\n", cartHeader.GetCartTypeName())
	fmt.Printf("ROM SIZE   %d KB\n", 32 << cartHeader.ROMSize)
	fmt.Printf("RAM SIZE   %s\n", cartHeader.GetRAMSize())
	fmt.Printf("DEST CODE  %s\n", cartHeader.GetDestCode())
	fmt.Printf("VERSION    %d\n", cartHeader.Version)
}

func main() {
	args := os.Args
	if len(args) < 2 {
		fmt.Println("Rom required")
		os.Exit(-1)
	}

	goboy := goboy{}
	goboy.Start()
	fmt.Printf("Loading %s\n", args[1])
	goboy.LoadCart(args[1])
}
