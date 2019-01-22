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
	"sort"
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

func displayRow(t *tracked) DisplayRow {
	row := DisplayRow{
		Status: "unknown",
		Name:   t.Target.Display(),
		Link:   t.Target.Link(),
	}

	if t.IsLive() {
		at := t.StartedAt().Format(time.Kitchen)
		row.Status = fmt.Sprintf("Now (%s)", at)
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

	return row
}

func displayList(lst []*tracked) (d []DisplayRow) {
	for _, t := range lst {
		d = append(d, displayRow(t))
	}

	return
}

// Display gives everyone we are tracking sorted by urgency for display by dashboard
func Display() (d DisplayTable) {
	m.RLock()
	defer m.RUnlock()

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
