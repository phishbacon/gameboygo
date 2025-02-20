package instructions

type Instruction struct {
  Mnemonic string
  Size uint8
  Cycles uint8
  AddressingMode string
  Z, N, H, C uint8 // 0 = reset, 1 = flag is set, 2 = flag untouched
}

var Instructions = map[uint8]Instruction{
  0x00: {
    Mnemonic: "NOP",
    Size: 1,
    Cycles: 4,
    Z: 2, N: 2, H: 2, C: 2,
  },
}
