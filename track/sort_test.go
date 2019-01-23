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
	"context"
	"sort"
	"testing"
	"time"
)

type dummy struct {
	name    string
	display string
	link    string
}

// BeginSnipe callback
func (t dummy) BeginSnipe() {
	return
}

// BeginSave callback
func (t dummy) BeginSave() {
	return
}

// EndSave callback
func (t dummy) EndSave(err error) {
	return
}

// Display for display in dashboard
func (t dummy) Display() string {
	return t.display
}

// Name is the streamers real name
func (t dummy) Name() string {
	return t.name
}

// Link is url string where this user's streams can be found
func (t dummy) Link() string {
	return t.link
}

// CheckLive gives true if the user is online
func (t dummy) CheckLive(context.Context) (bool, error) {
	return false, nil
}

// CheckStream gives nil if a stream has been found
func (t dummy) CheckStream(context.Context) (streamURL string, err error) {
	return "", nil
}

// SavePath gives nil if a stream has been found
func (t dummy) SavePath() string {
	return ""
}

func TestSort(t *testing.T) {
	a := dummy{
		name: "菅 原 早 記",
		link: "https://www.showroom-live.com/48_SUGAHARA_SAKI",
	}

	b := dummy{
		name: "齊 藤 京 子",
		link: "https://www.showroom-live.com/46_KYOKO_SAITO",
	}

	c := dummy{
		name: "田 口 愛 佳",
		link: "https://www.showroom-live.com/48_Manaka_Taguchi",
	}

	{
		got := []*tracked{
			&tracked{
				target: a,
			},
			&tracked{
				target: b,
			},
			&tracked{
				target: c,
			},
		}

		sort.Sort(byUrgency(got))
		// lexagraphical order is the fallback
		want := []*tracked{
			&tracked{
				target: c,
			},
			&tracked{
				target: a,
			},
			&tracked{
				target: b,
			},
		}

		for ndx := range got {
			if want[ndx].Name() != got[ndx].Name() {
				t.Error("want", want[ndx].Name(), "got", got[ndx].Name())
			}
		}
	}

	{
		got := []*tracked{
			&tracked{
				target: a,
			},
			&tracked{
				target: b,
			},
			&tracked{
				target: c,
			},
		}
		got[1].SetStartedAt(time.Now())
		got[2].SetUpcomingAt(time.Now().Add(15 * time.Minute))
		sort.Sort(byUrgency(got))

		want := []*tracked{
			&tracked{
				target: b,
			},
			&tracked{
				target: c,
			},
			&tracked{
				target: a,
			},
		}

		for ndx := range got {
			if want[ndx].Name() != got[ndx].Name() {
				t.Error("want", want[ndx].Name(), "got", got[ndx].Name())
			}
		}
	}
}
