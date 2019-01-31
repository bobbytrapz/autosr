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

type byUrgency []*tracked

func (s byUrgency) Len() int {
	return len(s)
}

func (s byUrgency) Swap(a, b int) {
	s[a], s[b] = s[b], s[a]
}

func (s byUrgency) Less(a, b int) bool {
	if s[a].IsLive() && !s[b].IsLive() {
		return true
	}

	if !s[a].IsLive() && s[b].IsLive() {
		return false
	}

	if s[a].IsLive() && s[b].IsLive() {
		aT := s[a].StartedAt()
		bT := s[b].StartedAt()
		if aT == bT {
			return s[a].Link() < s[b].Link()
		}
		return aT.After(bT)
	}

	if s[a].IsUpcoming() && !s[b].IsUpcoming() {
		return true
	}

	if !s[a].IsUpcoming() && s[b].IsUpcoming() {
		return false
	}

	if s[a].IsUpcoming() && s[b].IsUpcoming() {
		aT := s[a].UpcomingAt()
		bT := s[b].UpcomingAt()
		if aT == bT {
			return s[a].Link() < s[b].Link()
		}
		return aT.Before(bT)
	}

	if s[a].FinishedAt() != s[b].FinishedAt() {
		aT := s[a].FinishedAt()
		bT := s[b].FinishedAt()
		if aT == bT {
			return s[a].Link() < s[b].Link()
		}
		return aT.After(bT)
	}

	return s[a].Link() < s[b].Link()
}
