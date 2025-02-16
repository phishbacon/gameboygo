package cpu

type flagMask uint8

const (
	ZERO_FLAG        flagMask = 0b10000000
	SUBTRACTION_FLAG flagMask = 0b01000000
	HALF_CARRY_FLAG  flagMask = 0b00100000
	CARRY_FLAG       flagMask = 0b00010000
)

type Registers struct {
	A uint8
	B uint8
	C uint8
	D uint8
	E uint8
	F uint8 // Flags register znhc 0000 | Zero, Subtraction, Half Carry, Carry
	H uint8
	L uint8
}

// B = 00000001 C = 11110000 return 00000001 11110000
func (r Registers) GetBC() uint16 {
	return (uint16(r.B) << 8) | uint16(r.C)
}

// input = 00000001 11110000 B = 00000001, C = 11110000
func (r *Registers) SetBC(input uint16) {
	// B
	// 0000000111110000
	//&1111111100000000
	// 0000000100000000
	// >> 8
	// 0000000000000001
	// C
	// 0000000111110000
	//&0000000011111111
	// 0000000011110000
	r.B = uint8((input & 0xFF00) >> 8)
	r.C = uint8(input & 0x00FF)
}

// D = 00000001 E = 11110000 return 00000001 11110000
func (r Registers) GetDE() uint16 {
	return (uint16(r.D) << 8) | uint16(r.E)
}

// input = 00000001 11110000 D = 00000001, E = 11110000
func (r *Registers) SetDE(input uint16) {
	// D
	// 0000000111110000
	//&1111111100000000
	// 0000000100000000
	// >> 8
	// 0000000000000001
	// E
	// 0000000111110000
	//&0000000011111111
	// 0000000011110000
	r.D = uint8((input & 0xFF00) >> 8)
	r.E = uint8(input & 0x00FF)
}

// H = 00000001 L = 11110000 return 00000001 11110000
func (r Registers) GetHL() uint16 {
	return (uint16(r.H) << 8) | uint16(r.L)
}

// input = 00000001 11110000 H = 00000001, L = 11110000
func (r *Registers) SetHL(input uint16) {
	// H
	// 0000000111110000
	//&1111111100000000
	// 0000000100000000
	// >> 8
	// 0000000000000001
	// L
	// 0000000111110000
	//&0000000011111111
	// 0000000011110000
	r.H = uint8((input & 0xFF00) >> 8)
	r.L = uint8(input & 0x00FF)
}

// Returns true if the flag is flipped, false otherwise
func (r Registers) GetFlag(flag flagMask) bool {
  if (r.F & uint8(flag) > 0) {
    return true
  } else {
    return false
  }
}

// Flips the flag to true
func (r *Registers) SetFlag(flag flagMask) {
  r.F = r.F | uint8(flag)
}

// Flips the flag to false
func (r *Registers) UnsetFlag(flag flagMask) {
  // unset 0b01000000
  // F 11110000
  // & (10111111 & 11110000) = 10110000
  // F 10110000

  // & 0b11110000 to get rid of the last four bits we don't need from the
  // 1's compliment
  r.F = r.F & (^uint8(flag) & 0xF0)
}

type CPU struct {
	Registers Registers
}

func NewCPU() CPU {
	cpu := CPU{}
	return cpu
}
