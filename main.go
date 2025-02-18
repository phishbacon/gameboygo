package main

import (
	"fmt"
	"goboy/soc"
  "goboy/cart"
	"os"
)

type goboy struct {
  soc soc.SOC
  paused bool
  running bool
  ticks uint64
}

func (g *goboy) Start() {
  g.soc = soc.NewSOC()
  g.running = true
  g.paused = false
  g.ticks = 0
}

func main() {
  args := os.Args
  if (len(args) < 2) {
    fmt.Println("Rom required")
    os.Exit(-1)
  }

  goboy := goboy{}
  goboy.Start()
  
  for key, value := range cart.NewLicCodes {
    a := []rune(key)
    temp := uint16(a[0]) << 8
    temp = temp | uint16(a[1])
    fmt.Printf("%d: \"%s\",\n", temp, value)
  }
}

