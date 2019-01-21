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

// Run the dashboard
func Run() {
	// connect to server
	var err error
	remote, err = rpc.DialHTTP("tcp", "localhost:4846")
	if err != nil {
		fmt.Println("We cannot connect to the server. Try 'autosr shutdown' then try again.")
		return
	}

	// initialize tui
	g, err := gocui.NewGui(gocui.OutputNormal)
	if err != nil {
		panic(err)
	}
	defer g.Close()

	g.Cursor = true
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
	m.Lock()
	if err := remote.Call("Command.Status", req, &res); err != nil {
		// poll was not successful so close dashboard
		g.Update(func(g *gocui.Gui) error {
			return gocui.ErrQuit
		})

		m.Unlock()
		return
	}
	m.Unlock()

	// poll was successful so redraw
	g.Update(func(g *gocui.Gui) error {
		g.DeleteView("home")
		layout(g)

		return nil
	})
}

var logo = `
 ⢀⣀ ⡀⢀ ⣰⡀ ⢀⡀ ⢀⣀ ⡀⣀
 ⠣⠼ ⠣⠼ ⠘⠤ ⠣⠜ ⠭⠕ ⠏
`

var rainbow = `
 [0;1;35;95m⢀[0;1;31;91m⣀[0m [0;1;33;93m⡀⢀[0m [0;1;32;92m⣰[0;1;36;96m⡀[0m [0;1;34;94m⢀⡀[0m [0;1;35;95m⢀[0;1;31;91m⣀[0m [0;1;33;93m⡀⣀[0m
 [0;1;31;91m⠣[0;1;33;93m⠼[0m [0;1;32;92m⠣⠼[0m [0;1;36;96m⠘[0;1;34;94m⠤[0m [0;1;35;95m⠣⠜[0m [0;1;31;91m⠭[0;1;33;93m⠕[0m [0;1;32;92m⠏[0m
`

func layout(g *gocui.Gui) error {

	{
		w, h := g.Size()
		if v, err := g.SetView("home", -1, -1, w, h); err != nil {
			if err != gocui.ErrUnknownView {
				return err
			}

			v.Highlight = true
			v.Wrap = true
			v.Autoscroll = true

			fmt.Fprintln(v, logo)

			tw := tabwriter.NewWriter(v, 0, 0, 8, ' ', 0)
			if len(res.Tracking) > 0 {
				fmt.Fprintf(tw, "STATUS\tNAME\tURL\n")
			} else {
				fmt.Fprintln(tw, "use 'autosr track' to add targets.")
				fmt.Fprintln(tw, "For help visit: https://github.com/bobbytrapz/autosr")
			}
			var nowSep sync.Once
			var soonSep sync.Once
			sepFn := func() {
				fmt.Fprintln(tw, "\t\t\t")
			}
			for _, t := range res.Tracking {
				var status string
				if t.IsLive() {
					status = fmt.Sprintf("Now (%s)", t.StartedAt.Format(time.Kitchen))
				} else if t.IsUpcoming() {
					nowSep.Do(sepFn)
					status = fmt.Sprintf("Soon (%s)", time.Until(t.UpcomingAt).Truncate(time.Second))
				} else {
					soonSep.Do(sepFn)
					status = "Offline"
				}
				fmt.Fprintf(tw, "%s\t%s\t%s\n", status, t.Name, t.Link)
			}
			tw.Flush()
		}
	}

	return nil
}

func keys(g *gocui.Gui) (err error) {
	if err = g.SetKeybinding("", 'q', gocui.ModNone, quit); err != nil {
		return
	}

	if err = g.SetKeybinding("", gocui.KeyCtrlC, gocui.ModNone, quit); err != nil {
		return
	}

	if err = g.SetKeybinding("", gocui.KeyCtrlD, gocui.ModNone, quit); err != nil {
		return
	}

	if err = g.SetKeybinding("", gocui.KeyEnter, gocui.ModNone, selectURL); err != nil {
		return
	}

	return
}

func quit(g *gocui.Gui, v *gocui.View) error {
	return gocui.ErrQuit
}

func selectURL(g *gocui.Gui, v *gocui.View) error {
	m.Lock()
	defer m.Unlock()

	req.SelectURL = fmt.Sprintf("https://dummy.selection/t=%d", time.Now().Unix())
	redraw(g)

	return nil
}
