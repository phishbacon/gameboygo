package instructions

type AddrMode func() uint8
type Operation func()

type Instruction struct {
	mnemonic  string
	addrMode  AddrMode
	operation Operation
}
