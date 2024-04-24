package stepper

const Marker = '\x1B'

func IsTerminatorByte(c byte) bool {
	return (c >= 0x40 && c <= 0x5a) || (c >= 0x61 && c <= 0x7a)
}
