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

package backoff

import (
	"math/rand"
	"time"
)

func init() {
	rand.Seed(time.Now().UTC().UnixNano())
}

// Policy for retrying
type Policy struct {
	Steps []int
}

// DefaultPolicy backoff policy in ms
var DefaultPolicy = Policy{
	[]int{0, 10, 10, 100, 100, 500, 500, 3000, 3000, 5000, 5000, 10000, 10000, 20000, 20000, 40000, 40000},
}

// Duration gives how long we should wait on the given attempt
func (p *Policy) Duration(n int) time.Duration {
	if n >= len(p.Steps) {
		n = len(p.Steps) - 1
	}
	duration := p.Steps[n]
	if duration > 0 {
		// random int from uniform distribution in range of duration
		duration = duration/2 + rand.Intn(duration)
	}
	return time.Duration(duration) * time.Millisecond
}
