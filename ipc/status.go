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

package ipc

import (
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/bobbytrapz/autosr/track"
	"github.com/bobbytrapz/autosr/version"
)

// State of the dashboard
type State struct {
	Message        string
	Tracking       []track.Info
	SelectedTarget string
}

// DashboardClient represents a connected dashboard
type DashboardClient struct {
	SelectTarget string
}

var m sync.Mutex
var status = State{
	Message: fmt.Sprintf("autosr %s", version.String),
	Tracking: []track.Info{
		track.Info{
			Name: "菅 原 早 記",
			Link: "https://www.showroom-live.com/48_SUGAHARA_SAKI",
		},
		track.Info{
			Name: "齊 藤 京 子",
			Link: "https://www.showroom-live.com/46_KYOKO_SAITO",
		},
		track.Info{
			Name: "田 口 愛 佳",
			Link: "https://www.showroom-live.com/48_Manaka_Taguchi",
		},
	},
}

func init() {
	status.Tracking[1].StartedAt = time.Now()
	status.Tracking[2].UpcomingAt = time.Now().Add(15 * time.Minute)

	sort.Sort(track.ByUrgency(status.Tracking))
}

func replicate(dst *State) {
	dst.Message = status.Message
	dst.SelectedTarget = status.SelectedTarget

	dst.Tracking = make([]track.Info, len(status.Tracking))
	copy(dst.Tracking, status.Tracking)
}

// Status for the dashboard
func (c *Command) Status(client *DashboardClient, state *State) error {
	if client.SelectTarget != "" {
		status.SelectedTarget = client.SelectTarget
	}

	replicate(state)

	return nil
}
