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
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/bobbytrapz/autosr/backoff"
	"github.com/bobbytrapz/autosr/retry"
)

// SnipeTargetAt snipes a target at the given time
func SnipeTargetAt(t Target, at time.Time) error {
	if at.IsZero() {
		return errors.New("track.SnipeTargetAt: invalid time")
	}

	tracked, err := getTracked(t.Link())
	if err != nil {
		return fmt.Errorf("track.SnipeTarget: %s", err)
	}

	return SnipeAt(tracked, at)
}

// SnipeAt snipes a target at the given time
func SnipeAt(tracked *tracked, at time.Time) error {
	if at.IsZero() {
		return errors.New("track.SnipeAt: invalid time")
	}

	if tracked.Status() == saving {
		// already saving so we should not cancel it
		return nil
	}

	if tracked.Status() == sniping {
		// already sniping so this is redudant
		return nil
	}

	tracked.SetUpcomingAt(at)

	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	tracked.SetCancel(cancel)

	return snipe(ctx, tracked)
}

func snipe(ctx context.Context, tracked *tracked) error {
	if tracked.Status() == sniping {
		return nil
	}

	upcomingAt := tracked.UpcomingAt()

	if upcomingAt.IsZero() {
		return errors.New("track.snipe: invalid time")
	}

	tracked.SetStatus(sniping)
	tracked.Target.BeginSnipe()

	// snipe target
	go func() {
		wg.Add(1)
		defer wg.Done()

		// wait until we expect the target to stream
		check := time.NewTimer(time.Until(upcomingAt))
		defer check.Stop()

		for {
			select {
			case <-ctx.Done():
				log.Println("track.snipe:", tracked.Target.Name(), "canceled")
				tracked.SetStatus(sleeping)
				return
			case <-check.C:
				// set timeout for sniping
				timeout := time.NewTimer(15 * time.Minute)
				defer timeout.Stop()

				// check to see if the target's stream has actually begun
				url, err := tracked.Target.Check()
				if err != nil {
					if e, ok := retry.StringCheck(err); ok {
						// retry according to backoff policy
						for n := 0; ; n++ {
							select {
							case <-ctx.Done():
								log.Println("track.snipe:", tracked.Target.Name(), "canceled")
								tracked.SetStatus(sleeping)
								return
							case <-timeout.C:
								log.Println("track.snipe:", tracked.Target.Name(), "timeout")
								return
							case <-time.After(backoff.DefaultPolicy.Duration(n)):
								url, err = e.Retry()
								if err == nil {
									break
								}
								e, ok = retry.StringCheck(err)
								if !ok {
									// we failed and should not try again
									tracked.SetStatus(sleeping)
									return
								}
								log.Println("track.snipe:", err)
							}
						}
					}
				}
				// attempt ok
				log.Println("track.snipe:", tracked.Target.Name(), "found url.")
				tracked.SetStreamURL(url)
				tracked.SetUpcomingAt(time.Time{})
				if err := Save(ctx, tracked); err != nil {
					log.Println("track.snipe:", err)
				}
				return
			}
		}
	}()

	return nil
}
