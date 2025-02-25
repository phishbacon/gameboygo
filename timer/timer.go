package timer

type Timer struct {
	div  uint8
	tima uint8
	tma  uint8
	tac  uint8
}

func (t *Timer) Read(address uint16) uint8 {
  switch address {
  case 0xFF04:
    return t.div
  case 0xFF05:
    return t.tima
  case 0xFF06:
    return t.tma
  case 0xFF07:
    return t.tac
  }
  return 0
}

func (t *Timer) Write(address uint16, value uint8) {
  switch address {
  case 0xFF04:
    t.div = 0x0000
  case 0xFF05:
    t.tima = value
  case 0xFF06:
    t.tma = value
  case 0xFF07:
    t.tac = value
  }
}
