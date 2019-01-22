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
	"text/tabwriter"
	"time"

	"github.com/bobbytrapz/autosr/ipc"
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
	var err error
	remote, err = rpc.DialHTTP("tcp", "localhost:4846")
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

	numLive := len(res.TrackTable.Live)
	numUpcoming := len(res.TrackTable.Upcoming)

	tw := tabwriter.NewWriter(v, 0, 0, 4, ' ', 0)
	if numRows() == 0 {
		fmt.Fprintln(v, "Written by Bobby. (@pibisubukebe)")
		fmt.Fprintln(v, "use 'autosr track' to add targets.")
		fmt.Fprintln(v, "For help visit: https://github.com/bobbytrapz/autosr")

		return
	}
	v.SelBgColor = gocui.ColorGreen
	v.SelFgColor = gocui.ColorBlack

	for _, row := range res.TrackTable.Live {
		fmt.Fprintf(tw, "%s\t%s\t%s\n", row.Status, row.Name, row.Link)
	}
	if numLive > 0 {
		fmt.Fprintln(tw, "\t\t\t")
	}

	for _, row := range res.TrackTable.Upcoming {
		fmt.Fprintf(tw, "%s\t%s\t%s\n", row.Status, row.Name, row.Link)
	}
	if numUpcoming > 0 {
		fmt.Fprintln(tw, "\t\t\t")
	}

	for _, row := range res.TrackTable.Offline {
		fmt.Fprintf(tw, "%s\t%s\t%s\n", row.Status, row.Name, row.Link)
	}

	tw.Flush()
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
	if err = g.SetKeybinding("target-list", 'c', gocui.ModNone, cancelTarget); err != nil {
		return
	}

	if err = g.SetKeybinding("target-list", 'r', gocui.ModNone, reloadTargets); err != nil {
		return
	}

	return
}

func call(method string) error {
	m.Lock()
	defer m.Unlock()
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
	debug(fmt.Sprintf("cursor: %d %d", cx, cy-1))
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

	ox, oy := v.Origin()
	cx, cy := v.Cursor()
	if oy+cy+1 >= numRows() {
		return nil
	}
	debug(fmt.Sprintf("cursor: %d %d", cx, cy+1))
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

func readURL(g *gocui.Gui, v *gocui.View) error {
	_, cy := v.Cursor()
	if line, err := v.Line(cy); err == nil {
		ndx := strings.Index(line, "http")
		if ndx > -1 {
			req.SelectURL = line[ndx:]
		} else {
			req.SelectURL = ""
		}
	}

	debug(fmt.Sprintf("selected url: %s", req.SelectURL))
	return nil
}

func cancelTarget(g *gocui.Gui, v *gocui.View) error {
	readURL(g, v)

	if err := call("CancelTarget"); err != nil {
		return fmt.Errorf("dashboard.cancelTarget: %s", err)
	}
	redraw(g)
	return nil
}

func reloadTargets(g *gocui.Gui, v *gocui.View) error {
	if err := call("CheckNow"); err != nil {
		return fmt.Errorf("dashboard.reloadTargets: %s", err)
	}
	redraw(g)
	return nil
}
