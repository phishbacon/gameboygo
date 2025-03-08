package cpu

type FlagMask uint16

const (
	ZERO_FLAG        FlagMask = 0b10000000
	SUBTRACTION_FLAG FlagMask = 0b01000000
	HALF_CARRY_FLAG  FlagMask = 0b00100000
	CARRY_FLAG       FlagMask = 0b00010000
)

type Registers struct {
	A   uint8
	B   uint8
	C   uint8
	D   uint8
	E   uint8
	F   uint8
	H   uint8
	L   uint8
	SP  uint16 // Stack Pointer
	PC  uint16 // Program Counter
	IME bool   // Interrupt master enable
}

func (r *Registers) GetBC() uint16 {
	return ((uint16(r.B) << 8) & 0xFF00) | (uint16(r.C) & 0x00FF)
}

func (r *Registers) GetDE() uint16 {
	return ((uint16(r.D) << 8) & 0xFF00) | (uint16(r.E) & 0x00FF)
}

func (r *Registers) GetHL() uint16 {
	return ((uint16(r.H) << 8) & 0xFF00) | (uint16(r.L) & 0x00FF)
}

func (r *Registers) GetAF() uint16 {
	return ((uint16(r.A) << 8) & 0xFF00) | (uint16(r.F) & 0x00F0)
}

func (r *Registers) SetBC(value uint16) {
	lo := value & 0x00FF
	hi := (value >> 8) & 0x00FF

	r.B = uint8(hi)
	r.C = uint8(lo)
}

func (r *Registers) SetDE(value uint16) {
	lo := value & 0x00FF
	hi := (value >> 8) & 0x00FF

	r.D = uint8(hi)
	r.E = uint8(lo)
}

func (r *Registers) SetHL(value uint16) {
	lo := value & 0x00FF
	hi := (value >> 8) & 0x00FF

	r.H = uint8(hi)
	r.L = uint8(lo)
}

func (r *Registers) SetAF(value uint16) {
	lo := value & 0x00F0
	hi := (value >> 8) & 0x00FF

	r.A = uint8(hi)
	r.F = uint8(lo)
}

// Returns true if the flag is flipped, false otherwise
func (r *Registers) GetFlag(flag FlagMask) bool {
	if r.F&uint8(flag) > 0 {
		return true
	} else {
		return false
	}
}

// Sets the given flag to true or false
func (r *Registers) SetFlag(flag FlagMask, value bool) {
	if value {
		r.F = (r.F | uint8(flag)) & 0x00F0
	} else {
		r.F = (r.F & ^uint8(flag)) & 0x00F0
	}
}
