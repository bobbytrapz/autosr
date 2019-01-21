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
	"log"
	"time"

	"github.com/bobbytrapz/autosr/backoff"
	"github.com/bobbytrapz/autosr/options"
	"github.com/bobbytrapz/autosr/retry"
)

const retryAttempts = 3

var check = make(chan int, 1)

// CheckNow makes poll process right now
func CheckNow() {
	check <- 1
}

// Poll allows modules to monitor a website
func Poll(ctx context.Context, pollfn func() error) error {
	attempt := func() {
		err := pollfn()
		if err != nil {
			// retry if possible
			e, ok := retry.Check(err)
			numAttempts := 0
			for ; ok; e, ok = retry.Check(err) {
				select {
				case <-ctx.Done():
					return
				case <-time.After(backoff.DefaultPolicy.Duration(numAttempts)):
					numAttempts++
					err = e.Retry()
					if err == nil {
						break
					}
					log.Println("track.Poll:", err)
				}
			}
		}
	}

	// make first attempt right away
	attempt()

	// poll
	go func() {
		defer close(check)

		pollRate := options.GetDuration("check_every")
		log.Println("track.Poll:", pollRate)
		tick := time.NewTicker(pollRate)
		defer tick.Stop()
		for {
			select {
			case <-ctx.Done():
				log.Println("track.Poll: done")
				return
			case <-tick.C:
			case <-check:
				attempt()
				// check if poll rate was adjusted
				p := options.GetDuration("check_every")
				if p != pollRate {
					pollRate = p
					tick.Stop()
					tick = time.NewTicker(pollRate)
					log.Println("track.Poll: new poll rate", pollRate)
				}
			}
		}
	}()

	return nil
}
