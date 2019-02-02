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

package track

import (
	"fmt"
	"io"
	"log"
	"sort"
	"strings"
	"text/tabwriter"
	"time"
)

// DisplayRow of data
type DisplayRow struct {
	Status string
	Name   string
	Link   string
}

// DisplayTable tracking data
type DisplayTable struct {
	Live     []DisplayRow
	Upcoming []DisplayRow
	Offline  []DisplayRow
}

func displayRow(t *tracked) (row DisplayRow, err error) {
	row = DisplayRow{
		Status: "unknown",
		Name:   t.Display(),
		Link:   t.Link(),
	}

	if row.Name == "" || row.Link == "" {
		err = fmt.Errorf("track.displayRow: invalid row")
		return
	}

	if t.IsLive() {
		d := time.Now().Sub(t.StartedAt()).Truncate(5 * time.Minute)
		if d > time.Second {
			s := strings.TrimSuffix(d.String(), "0s")
			row.Status = fmt.Sprintf("Now (%s)", s)
		} else {
			row.Status = "Now"
		}
	} else if t.IsUpcoming() {
		at := time.Until(t.UpcomingAt()).Truncate(time.Second)
		if at > time.Second {
			row.Status = fmt.Sprintf("Soon (%s)", at)
		} else {
			row.Status = "Soon"
		}
	} else if t.IsOffline() {
		row.Status = "Offline"
	}

	return
}

func displayList(lst []*tracked) (d []DisplayRow) {
	for _, t := range lst {
		row, err := displayRow(t)
		if err == nil {
			d = append(d, row)
		} else {
			log.Println("track.displayList:", err)
		}
	}

	return
}

// Display gives everyone we are tracking sorted by urgency for display by dashboard
func Display() (d DisplayTable) {
	rw.RLock()
	defer rw.RUnlock()

	var live []*tracked
	var upcoming []*tracked
	var offline []*tracked
	for _, t := range tracking {
		if t.IsLive() {
			live = append(live, t)
		} else if t.IsUpcoming() {
			upcoming = append(upcoming, t)
		} else if t.IsOffline() {
			offline = append(offline, t)
		}
	}

	sort.Sort(byUrgency(live))
	sort.Sort(byUrgency(upcoming))
	sort.Sort(byUrgency(offline))

	d.Live = displayList(live)
	d.Upcoming = displayList(upcoming)
	d.Offline = displayList(offline)

	return
}

// Output for ui
func (d DisplayTable) Output(dst io.Writer) error {
	tw := tabwriter.NewWriter(dst, 0, 0, 4, ' ', 0)

	for _, row := range d.Live {
		fmt.Fprintf(tw, "%s\t%s\n", row.Status, row.Name)
	}
	if len(d.Live) > 0 {
		fmt.Fprintln(tw, "\t\t\t")
	}

	for _, row := range d.Upcoming {
		fmt.Fprintf(tw, "%s\t%s\n", row.Status, row.Name)
	}
	if len(d.Upcoming) > 0 {
		fmt.Fprintln(tw, "\t\t\t")
	}

	for _, row := range d.Offline {
		fmt.Fprintf(tw, "%s\t%s\n", row.Status, row.Name)
	}

	return tw.Flush()
}
