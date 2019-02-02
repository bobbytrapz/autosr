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
	"fmt"
	"net/rpc"
	"strings"
	"sync"
	"time"

	"github.com/bobbytrapz/autosr/ipc"
	"github.com/bobbytrapz/autosr/options"
	"github.com/bobbytrapz/autosr/track"
	"github.com/jroimartin/gocui"
)

var m sync.Mutex
var remote *rpc.Client
var req = ipc.Dashboard{
	SelectURL: "?",
}
var res ipc.Dashboard

var shouldColorLogo = false

var logoHeight = 2

// smbraille
var logo = `
 ⢀⣀ ⡀⢀ ⣰⡀ ⢀⡀ ⢀⣀ ⡀⣀
 ⠣⠼ ⠣⠼ ⠘⠤ ⠣⠜ ⠭⠕ ⠏
`

var colorLogo = `
 [0;1;35;95m⢀[0;1;31;91m⣀[0m [0;1;33;93m⡀⢀[0m [0;1;32;92m⣰[0;1;36;96m⡀[0m [0;1;34;94m⢀⡀[0m [0;1;35;95m⢀[0;1;31;91m⣀[0m [0;1;33;93m⡀⣀[0m
 [0;1;31;91m⠣[0;1;33;93m⠼[0m [0;1;32;92m⠣⠼[0m [0;1;36;96m⠘[0;1;34;94m⠤[0m [0;1;35;95m⠣⠜[0m [0;1;31;91m⠭[0;1;33;93m⠕[0m [0;1;32;92m⠏[0m
`

func debug(s string) {
	none := struct{}{}
	if err := remote.Call("Command.Debug", &s, &none); err != nil {
		panic(err)
	}
}

// Run the dashboard
func Run(bColor bool) {
	shouldColorLogo = bColor
	// connect to server
	addr := options.Get("listen_on")
	var err error
	remote, err = rpc.DialHTTP("tcp", addr)
	if err != nil {
		fmt.Println("We cannot connect to the server. Try 'autosr stop' then try again.")
		return
	}

	// initialize tui
	g, err := gocui.NewGui(gocui.OutputNormal)
	if err != nil {
		panic(err)
	}
	defer g.Close()

	g.Mouse = true
	g.Highlight = true

	g.SetManagerFunc(layout)

	if err := keys(g); err != nil {
		panic(err)
	}

	// poll server for dashboard updates
	poll := time.NewTicker(1 * time.Second)
	defer poll.Stop()
	go func() {
		for {
			select {
			case <-poll.C:
				redraw(g)
			}
		}
	}()

	// loop
	if err := g.MainLoop(); err != nil && err != gocui.ErrQuit {
		panic(err)
	}
}

func redraw(g *gocui.Gui) {
	if err := call("Status"); err != nil {
		g.Update(func(g *gocui.Gui) error {
			return gocui.ErrQuit
		})
	}

	g.Update(func(g *gocui.Gui) error {
		if v := g.CurrentView(); v != nil {
			switch v.Name() {
			case "target-list":
				drawTargetList(v)
				// fix cursor
				_, cy := v.Cursor()
				if l, err := v.Line(cy); err == nil && strings.TrimSpace(l) == "" {
					return moveUp(g, v)
				}
			}
		}

		return nil
	})
}

func drawLogo(v *gocui.View) {
	v.Clear()

	if shouldColorLogo {
		fmt.Fprintf(v, colorLogo)
	} else {
		fmt.Fprintf(v, logo)
	}
}

func drawTargetList(v *gocui.View) {
	v.Clear()
	v.SelBgColor = 0
	v.SelFgColor = 0

	if numRows() == 0 {
		fmt.Fprintln(v, "Written by Bobby. (@pibisubukebe)")
		fmt.Fprintln(v, "use 'autosr track' to add targets.")
		fmt.Fprintln(v, "For help visit: https://github.com/bobbytrapz/autosr")

		return
	}
	v.SelBgColor = colorFromString(options.Get("select_fg_color"))
	v.SelFgColor = colorFromString(options.Get("select_bg_color"))

	// write display to view
	res.TrackTable.Output(v)
}

func layout(g *gocui.Gui) error {
	w, h := g.Size()
	if v, err := g.SetView("logo", -1, -1, w, logoHeight); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}

		drawLogo(v)
	}

	if v, err := g.SetView("target-list", -1, logoHeight, w, h); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}

		v.Highlight = true

		drawTargetList(v)
	}

	if _, err := g.SetCurrentView("target-list"); err != nil {
		return err
	}

	return nil
}

func keys(g *gocui.Gui) (err error) {
	// quit
	if err = g.SetKeybinding("", 'q', gocui.ModNone, quit); err != nil {
		return
	}

	if err = g.SetKeybinding("", gocui.KeyCtrlC, gocui.ModNone, quit); err != nil {
		return
	}

	if err = g.SetKeybinding("", gocui.KeyCtrlD, gocui.ModNone, quit); err != nil {
		return
	}

	// cursor select
	if err = g.SetKeybinding("target-list", gocui.KeyArrowUp, gocui.ModNone, moveUp); err != nil {
		return
	}

	if err = g.SetKeybinding("target-list", gocui.KeyArrowDown, gocui.ModNone, moveDown); err != nil {
		return
	}

	// command
	if err = g.SetKeybinding("target-list", 'r', gocui.ModNone, reloadTargets); err != nil {
		return
	}

	if err = g.SetKeybinding("target-list", 'o', gocui.ModNone, openTarget); err != nil {
		return
	}

	// mouse
	if err = g.SetKeybinding("target-list", gocui.MouseRight, gocui.ModNone, openTarget); err != nil {
		return
	}

	return
}

func call(method string) error {
	m.Lock()
	defer m.Unlock()

	// clear tables
	res.TrackTable.Live = nil
	res.TrackTable.Upcoming = nil
	res.TrackTable.Offline = nil

	if err := remote.Call("Command."+method, req, &res); err != nil {
		return fmt.Errorf("dashboard.call: %s", err)
	}

	return nil
}

func quit(g *gocui.Gui, v *gocui.View) error {
	return gocui.ErrQuit
}

func numRows() int {
	return len(res.TrackTable.Live) + len(res.TrackTable.Upcoming) + len(res.TrackTable.Offline)
}

func moveUp(g *gocui.Gui, v *gocui.View) error {
	if v == nil {
		return nil
	}

	ox, oy := v.Origin()
	cx, cy := v.Cursor()
	if err := v.SetCursor(cx, cy-1); err != nil && oy > 0 {
		if err := v.SetOrigin(ox, oy-1); err != nil {
			debug(fmt.Sprintf("origin: %d %d", ox, oy-1))
			return err
		}
	}
	if l, err := v.Line(cy - 1); err == nil && strings.TrimSpace(l) == "" {
		return moveUp(g, v)
	}

	return nil
}

func moveDown(g *gocui.Gui, v *gocui.View) error {
	if v == nil {
		return nil
	}

	numSeparators := 0
	if len(res.TrackTable.Live) > 0 {
		numSeparators++
	}
	if len(res.TrackTable.Upcoming) > 0 {
		numSeparators++
	}
	ox, oy := v.Origin()
	cx, cy := v.Cursor()
	if oy+cy-numSeparators+1 > numRows() {
		return nil
	}

	if err := v.SetCursor(cx, cy+1); err != nil {
		if err := v.SetOrigin(ox, oy+1); err != nil {
			debug(fmt.Sprintf("origin: %d %d", ox, oy+1))
			return err
		}
	}
	if l, err := v.Line(cy + 1); err == nil && strings.TrimSpace(l) == "" {
		return moveDown(g, v)
	}

	return nil
}

func selected(v *gocui.View) (row track.DisplayRow) {
	_, cy := v.Cursor()
	line, err := v.Line(cy)
	if err != nil {
		return
	}

	sp := strings.SplitN(line, " ", 3)
	var name string
	if len(sp) == 2 {
		name = sp[1]
	} else if len(sp) == 3 {
		name = sp[2]
	} else {
		return
	}
	name = strings.TrimLeft(name, " ")

	for _, row := range res.TrackTable.Live {
		if row.Name == name {
			return row
		}
	}

	for _, row := range res.TrackTable.Upcoming {
		if row.Name == name {
			return row
		}
	}

	for _, row := range res.TrackTable.Offline {
		if row.Name == name {
			return row
		}
	}

	return
}

func openTarget(g *gocui.Gui, v *gocui.View) error {
	row := selected(v)
	openLink(row.Link)
	return nil
}

func reloadTargets(g *gocui.Gui, v *gocui.View) error {
	if err := call("CheckNow"); err != nil {
		return fmt.Errorf("dashboard.reloadTargets: %s", err)
	}
	redraw(g)
	return nil
}
