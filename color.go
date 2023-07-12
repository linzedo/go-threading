package threading

import "fmt"

const (
	Black Color = iota + 30
	Red
	Green
	Yellow
	Blue
	Magenta
	Cyan
	White
)

type Color uint8

var isColor bool

func SetColor(b bool) {
	isColor = b
}
func (c Color) Add(s string) string {
	if isColor {
		return fmt.Sprintf("\x1b[%dm%s\x1b[0m", uint8(c), s)
	}
	return s
}
