package font

import (
	"image/color"
	"strings"

	"github.com/hajimehoshi/bitmapfont/v3"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"
)

const LineHeight = 16

func tokens(str string) []string {
	tokens := []string{}
	for {
		var strs []string
		switch len(tokens) % 2 {
		case 0:
			strs = strings.SplitN(str, "<red>", 2)
		case 1:
			strs = strings.SplitN(str, "</red>", 2)
		}
		if len(strs) >= 1 {
			tokens = append(tokens, strs[0])
		}
		if len(strs) == 2 {
			str = strs[1]
		} else {
			break
		}
	}
	return tokens
}

func Width(str string) int {
	w := fixed.I(0)
	for _, t := range tokens(str) {
		w += font.MeasureString(bitmapfont.Face, t)
	}
	return w.Round()
}

func Height(str string) int {
	return (strings.Count(str, "\n") + 1) * LineHeight
}

var red = color.RGBA{0xe4, 0x32, 0x60, 0xff}

var face = text.NewGoXFace(bitmapfont.Face)

func DrawText(target *ebiten.Image, str string, x, y int, clr color.Color) {
	xf := float64(x)
	yf := float64(y)
	for i, t := range tokens(str) {
		clr := clr
		if i%2 == 1 {
			clr = red
		}
		op := &text.DrawOptions{}
		op.GeoM.Translate(xf, yf)
		op.ColorScale.ScaleWithColor(clr)
		text.Draw(target, t, face, op)
		xf += text.Advance(t, face)
	}
}
