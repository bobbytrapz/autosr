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

// Poll allows modules to monitor a website
func Poll(pollfn func() error) (cancel context.CancelFunc, err error) {
	ctx := context.Background()
	ctx, cancel = context.WithCancel(ctx)

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

		if err != nil {
			return
		}

		// attempt ok
		log.Println("track.Poll: ok")
		m.RLock()
		defer m.RUnlock()
		for link, tracked := range tracking {
			status := tracked.Status()

			log.Println("track.Poll: check", link)
			if status == sleeping && tracked.IsUpcoming() {
				tracked.Lock()
				// set up a context for this target
				ctx := context.Background()
				ctx, cancel := context.WithCancel(ctx)
				tracked.SetCancel(cancel)

				// begin sniping the target stream
				if err := Snipe(ctx, tracked); err != nil {
					log.Println("track.Poll:", err)
				}
				tracked.Unlock()
			}
		}
	}

	// make first attempt right away
	attempt()

	// poll
	go func() {
		pollRate := options.GetDuration("poll_rate")
		log.Println("track.Poll:", pollRate)
		tick := time.NewTicker(pollRate)
		defer tick.Stop()
		for {
			select {
			case <-ctx.Done():
				log.Println("track.Poll: done")
				return
			case <-tick.C:
				attempt()
				// check if poll rate was adjusted
				p := options.GetDuration("poll_rate")
				if p != pollRate {
					pollRate = p
					tick.Stop()
					tick = time.NewTicker(pollRate)
					log.Println("track.Poll: new poll rate", pollRate)
				}
			}
		}
	}()

	return
}
