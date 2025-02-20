package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"goboy/cart"
	"goboy/soc"
	"goboy/util"
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
	g.soc = soc.NewSOC(&g.cart)
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
	fmt.Printf("ROM SIZE   %d KB\n", 32<<cartHeader.ROMSize)
	fmt.Printf("RAM SIZE   %s\n", cartHeader.GetRAMSize())
	fmt.Printf("DEST CODE  %s\n", cartHeader.GetDestCode())
	fmt.Printf("VERSION    %d\n", cartHeader.Version)

	var checksum uint8 = 0
	for address := 0x0134; address <= 0x014C; address++ {
		checksum = checksum - g.cart[address] - 1
	}

	checksumPassed := util.If(cartHeader.HeaderChecksum == (checksum&0x00FF), "PASSED", "FAILED")
	fmt.Printf("CHECKSUM   %s\n", checksumPassed)
	util.DumpHex(g.cart)
}

func main() {
	args := os.Args
	if len(args) < 2 {
		fmt.Println("Rom required")
		os.Exit(-1)
	}

	goboy := goboy{}
	goboy.LoadCart(args[1])
	goboy.Start()
	fmt.Printf("Loading %s\n", args[1])

	// for goboy.running {
	//   if (goboy.paused) {
	//     continue;
	//   }
	//
	//   goboy.ticks++
	// }
}
