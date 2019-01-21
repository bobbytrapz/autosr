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
	"sort"
	"testing"
	"time"
)

func TestSortByUrgency(t *testing.T) {
	a := Info{
		Name: "菅 原 早 記",
		Link: "https://www.showroom-live.com/48_SUGAHARA_SAKI",
	}

	b := Info{
		Name: "齊 藤 京 子",
		Link: "https://www.showroom-live.com/46_KYOKO_SAITO",
	}

	c := Info{
		Name: "田 口 愛 佳",
		Link: "https://www.showroom-live.com/48_Manaka_Taguchi",
	}

	{
		got := []Info{a, b, c}
		sort.Sort(byUrgency(got))
		// lexagraphical order is the fallback
		want := []Info{c, a, b}
		for ndx := range got {
			if want[ndx].Name != got[ndx].Name {
				t.Error("want", want[ndx], "got", got[ndx])
			}
		}
	}

	{
		got := []Info{a, b, c}
		got[1].StartedAt = time.Now()
		got[2].UpcomingAt = time.Now().Add(15 * time.Minute)
		sort.Sort(byUrgency(got))
		want := []Info{b, c, a}
		for ndx := range got {
			if want[ndx].Name != got[ndx].Name {
				t.Error("want", want[ndx], "got", got[ndx])
			}
		}
	}

}
