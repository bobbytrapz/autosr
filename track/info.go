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
	"time"
)

// Info information about target for display
type Info struct {
	Name       string
	Link       string
	UpcomingAt time.Time
	StartedAt  time.Time
	FinishedAt time.Time
}

// IsUpcoming is true if stream has a known upcoming time
func (i Info) IsUpcoming() bool {
	return time.Until(i.UpcomingAt) > 0
}

// IsLive is true if stream is active
func (i Info) IsLive() bool {
	return !i.StartedAt.IsZero() && !i.IsFinished()
}

// IsFinished is true if stream has ended
func (i Info) IsFinished() bool {
	return !i.FinishedAt.IsZero()
}

// ByUrgency provides sorting based on target urgency
type ByUrgency []Info

func (s ByUrgency) Len() int {
	return len(s)
}

func (s ByUrgency) Swap(a, b int) {
	s[a], s[b] = s[b], s[a]
}

func (s ByUrgency) Less(a, b int) bool {
	if s[a].IsLive() && !s[b].IsLive() {
		return true
	}

	if !s[a].IsLive() && s[b].IsLive() {
		return false
	}

	if s[a].IsLive() && s[b].IsLive() {
		return s[a].StartedAt.Before(s[b].StartedAt)
	}

	if s[a].IsUpcoming() && !s[b].IsUpcoming() {
		return true
	}

	if !s[a].IsUpcoming() && s[b].IsUpcoming() {
		return false
	}

	if s[a].IsUpcoming() && s[b].IsUpcoming() {
		return s[a].UpcomingAt.Before(s[b].UpcomingAt)
	}

	return s[a].Name < s[b].Name
}
