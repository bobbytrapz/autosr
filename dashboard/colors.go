// This file is part of autosr.
//
// autosr is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// autosr is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with autosr.  If not, see <https://www.gnu.org/licenses/>.

package dashboard

import (
	"github.com/jroimartin/gocui"
	"strings"
)

func colorFromString(c string) (attr gocui.Attribute) {
	c = strings.ToLower(c)
	switch c {
	case "black":
		attr = gocui.ColorBlack
	case "red":
		attr = gocui.ColorRed
	case "green":
		attr = gocui.ColorGreen
	case "yellow":
		attr = gocui.ColorYellow
	case "blue":
		attr = gocui.ColorBlue
	case "magenta":
		attr = gocui.ColorMagenta
	case "cyan":
		attr = gocui.ColorCyan
	case "white":
		attr = gocui.ColorWhite
	default:
		attr = gocui.ColorDefault
	}

	return
}
