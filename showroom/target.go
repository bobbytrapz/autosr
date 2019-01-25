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

package showroom

import (
	"context"
	"fmt"
	"log"
	"path"
	"time"

	"github.com/bobbytrapz/autosr/retry"
	"github.com/bobbytrapz/autosr/track"
)

type target struct {
	// info
	name    string
	display string
	id      int
	link    string
	urlKey  string
}

// BeginSnipe callback
func (t target) BeginSnipe() {
	log.Println("showroom.BeginSnipe:", t.name)
	return
}

// BeginSave callback
func (t target) BeginSave() {
	log.Println("showroom.BeginSave:", t.name)
	return
}

// EndSave callback
func (t target) EndSave() {
	log.Println("showroom.EndSave:", t.name)
	return
}

// Display for display in dashboard
func (t target) Display() string {
	return t.display
}

// Name is the streamers real name
func (t target) Name() string {
	return t.name
}

// Link is url string where this user's streams can be found
func (t target) Link() string {
	return t.link
}

// CheckLive gives true if the user is online
func (t target) CheckLive(ctx context.Context) (isLive bool, err error) {
	// check to see if the user is live
	isLive, err = checkIsLive(ctx, t.id)
	if err == nil && !isLive {
		err = retry.BoolError{
			Message: fmt.Sprintf("%s is not live yet", t.name),
			Attempt: func() (bool, error) {
				return t.CheckLive(ctx)
			},
		}
	}

	return
}

// CheckStream gives nil if a stream has been found and expects the user to possibly be live
func (t target) CheckStream(ctx context.Context) (streamURL string, err error) {
	// check for stream
	if s, err := checkStreamURL(ctx, t.id); err == nil && s != "" {
		return s, nil
	}

	// check for upcoming time
	var at time.Time
	if at, err = checkNextLive(ctx, t.id); err == nil && !at.IsZero() {
		// there's a date set so maybe add a snipe
		if err = track.SnipeTargetAt(ctx, t, at); err != nil {
			log.Println("showroom.CheckStream:", err)

			return
		}

		err = fmt.Errorf("%s has a new upcoming time set", t.name)

		return
	}

	err = retry.StringError{
		Message: fmt.Sprintf("%s has no stream yet", t.name),
		Attempt: func() (string, error) {
			return t.CheckStream(ctx)
		},
	}

	return
}

// SavePath decides where videos are saved
func (t target) SavePath() string {
	if t.name != "" {
		return t.name
	}

	return path.Base(t.link)
}
