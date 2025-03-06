package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"goboy/cart"
	"goboy/display"
	"goboy/soc"
	"goboy/util"
	"os"
	"runtime"
	"strconv"

	qt "github.com/mappu/miqt/qt6"
)

func init() {
	runtime.LockOSThread()
}

type Goboy struct {
	soc     *soc.SOC
	cart    *cart.Cart
	on      bool
	display *display.Display
}

func NewGoboy(args []string) *Goboy {
	qt.NewQApplication(args)

	goboy := &Goboy{}
	dis := display.NewDisplay()
	soc := soc.NewSOC()
	goboy.soc = soc
	goboy.on = false
	goboy.display = dis

	dis.Window.Show()
	dis.Widget.SetFixedWidth(1000)
	dis.Window.OnCloseEvent(func(super func(event *qt.QCloseEvent), event *qt.QCloseEvent) {
		goboy.on = false
		event.Accept() // Allow the window to close (use event.Ignore() to prevent closing)
		super(event)
	})

	// Layout the ui
	btn := qt.NewQPushButton3("Step")
	stepInput := qt.NewQLineEdit2()
	btn2 := qt.NewQPushButton3("Pause")
	currStepValues := qt.NewQPlainTextEdit2()
	goboy.display.InstrText = currStepValues
	currStepValues.SetReadOnly(true)

	btn.OnPressed(func() {
		steps := stepInput.Text()
		if steps == "" {
			goboy.soc.Step(1)
		} else {
			stepsInt, err := strconv.Atoi(steps)
			if err != nil {
				panic(err)
			}
			goboy.soc.Step(stepsInt)
		}
	})
	dis.Layout.AddWidget2(btn.QWidget, 0, 0)

	stepInput.SetPlaceholderText("How many?")
	intValidator := qt.NewQIntValidator2(0, 100)
	stepInput.SetValidator(intValidator.QValidator)
	dis.Layout.AddWidget2(stepInput.QWidget, 0, 1)

	btn2.OnPressed(func() {
		goboy.soc.Paused = !goboy.soc.Paused
	})
	dis.Layout.AddWidget2(btn2.QWidget, 1, 0)

	dis.Layout.AddWidget2(currStepValues.QWidget, 3, 0)
	dis.Layout.SetColumnStretch(0, 3)
	dis.Layout.SetRowStretch(3, 1)
	return goboy
}

func (g *Goboy) Start() {
	g.on = true
	go g.soc.Init()
	qt.QApplication_Exec()
}

func (g *Goboy) LoadCart(fileName string) {
	dump, dumpErr := os.ReadFile(fileName)
	if dumpErr != nil {
		fmt.Print(dumpErr)
		os.Exit(-1)
	}
	// give bus reference to cart so soc components can read from it
	g.cart = (*cart.Cart)(&dump)
	g.soc.Bus.ConnectCart(&dump)

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

	goboy := NewGoboy(args)
	if len(args) < 2 {
		fmt.Println("Rom required")
		os.Exit(-1)
	}

	goboy.LoadCart(args[1])

	fmt.Printf("Loading %s\n", args[1])
	if goboy.cart.VerifyLogoDump() {
		// jump to address 0x0100
	} else {
		fmt.Println("Failed to verify logo")
		os.Exit(-1)
	}

	goboy.Start()

	for goboy.on {
	}
}
