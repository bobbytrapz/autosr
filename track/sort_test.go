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
	name string
	link string
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
func (t dummy) EndSave() {
	return
}

// Display for display in dashboard
func (t dummy) Display() string {
	return t.name
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

func TestSortByUrgency(t *testing.T) {
	a := dummy{
		name: "菅 原 早 記",
		link: "https://www.showroom-live.com/48_SUGAHARA_SAKI",
	}

	b := dummy{
		name: "田 口 愛 佳",
		link: "https://www.showroom-live.com/48_Manaka_Taguchi",
	}

	c := dummy{
		name: "齊 藤 京 子",
		link: "https://www.showroom-live.com/46_KYOKO_SAITO",
	}

	t.Run("all targets have the same time", func(t *testing.T) {
		got := []*tracked{
			{
				target: a,
			},
			{
				target: b,
			},
			{
				target: c,
			},
		}

		sort.Sort(byUrgency(got))
		want := []*tracked{
			{
				target: c,
			},
			{
				target: b,
			},
			{
				target: a,
			},
		}

		for ndx := range got {
			if want[ndx].Name() != got[ndx].Name() {
				t.Error("want", want[ndx].Name(), "got", got[ndx].Name())
			}
		}
	})

	t.Run("two targets have upcoming times", func(t *testing.T) {
		got := []*tracked{
			{
				target: a,
			},
			{
				target: b,
			},
			{
				target: c,
			},
		}
		now := time.Now()

		atask := snipeTask{
			name: a.Name(),
			link: a.Link(),
			at:   now,
		}
		addSnipeTask(atask)
		defer delSnipeTask(atask)

		ctask := snipeTask{
			name: c.Name(),
			link: c.Link(),
			at:   now.Add(15 * time.Minute),
		}
		addSnipeTask(ctask)
		defer delSnipeTask(ctask)

		diff := got[2].UpcomingAt().Sub(got[0].UpcomingAt())
		if diff != 15*time.Minute {
			t.Error("want", 15*time.Minute, "got", diff)
		}

		sort.Sort(byUrgency(got))

		want := []*tracked{
			{
				target: a,
			},
			{
				target: c,
			},
			{
				target: b,
			},
		}

		for ndx := range got {
			if want[ndx].Name() != got[ndx].Name() {
				t.Error("want", want[ndx].Name(), "got", got[ndx].Name())
			}
		}
	})
}

func TestSortUpcomingSameTime(t *testing.T) {
	// bug: in the dashboard targets with the same upcoming time
	// did not always appear in the same order
	// so the names jumped around
	a := &tracked{
		target: dummy{
			name: "菅 原 早 記",
			link: "https://www.showroom-live.com/48_SUGAHARA_SAKI",
		},
	}

	b := &tracked{
		target: dummy{
			name: "齊 藤 京 子",
			link: "https://www.showroom-live.com/46_KYOKO_SAITO",
		},
	}

	c := &tracked{
		target: dummy{
			name: "田 口 愛 佳",
			link: "https://www.showroom-live.com/48_Manaka_Taguchi",
		},
	}

	at := time.Now()

	// a is coming up a bit later so should always be last
	atask := snipeTask{
		name: a.Name(),
		link: a.Link(),
		at:   at.Add(10 * time.Minute),
	}
	addSnipeTask(atask)
	defer delSnipeTask(atask)

	// b and c are coming on at the same time
	// one should consistently be displayed before the other
	btask := snipeTask{
		name: b.Name(),
		link: b.Link(),
		at:   at,
	}
	addSnipeTask(btask)
	defer delSnipeTask(btask)

	ctask := snipeTask{
		name: c.Name(),
		link: c.Link(),
		at:   at,
	}
	addSnipeTask(ctask)
	defer delSnipeTask(ctask)

	tracking[a.Link()] = a
	tracking[b.Link()] = b
	tracking[c.Link()] = c

	got := Display()
	want := []*tracked{b, c, a}

	for ndx := range got.Upcoming {
		if want[ndx].Name() != got.Upcoming[ndx].Name {
			t.Error("want", want[ndx].Name(), "got", got.Upcoming[ndx].Name)
		}
	}
}
