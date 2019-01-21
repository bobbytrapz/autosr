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
	"fmt"
	"log"
	"net/url"
	"path"
	"time"

	"github.com/bobbytrapz/autosr/retry"
	"github.com/bobbytrapz/autosr/track"
)

// AddTargetFromURL adds showroom user using the url
func AddTargetFromURL(link string) error {
	_, err := url.Parse(link)
	if err != nil {
		return fmt.Errorf("showroom.AddTargetFromURL: %s", err)
	}

	s, err := fetchRoom(link)
	if err != nil {
		return fmt.Errorf("showroom.AddTargetFromURL: %s", err)
	}

	t := Target{
		name:   s.Name,
		id:     s.ID,
		link:   link,
		urlKey: s.LiveRoom.URLKey,
	}

	err = track.AddTarget(t)
	if err != nil {
		// we must be already targeting
		return nil
	}

	m.Lock()
	targets = append(targets, t)
	m.Unlock()

	// check target right away
	if streamURL, err := t.Check(); err == nil {
		log.Println("showroom.AddTargetFromURL:", t.name, "is live now!", streamURL)
		// they are live now so snipe them now
		track.SnipeTargetAt(t, time.Now())

		return nil
	}

	log.Println("showroom.AddTargetFromURL:", t.name, err)

	return nil
}

// RemoveTargetFromURL removes showroom user using the url
func RemoveTargetFromURL(link string) error {
	if err := track.RemoveTarget(link); err != nil {
		return fmt.Errorf("showroom.RemoveTargetFromURL: %s", err)
	}

	m.Lock()
	n := 0
	for ; n < len(targets); n++ {
		if targets[n].link == link {
			break
		}
	}
	if n < len(targets) {
		targets = append(targets[:n], targets[n+1:]...)
	}
	m.Unlock()

	return nil
}

// Target showroom streamer
type Target struct {
	// info
	name   string
	id     int
	link   string
	urlKey string
}

// BeginSnipe callback
func (t Target) BeginSnipe() {
	log.Println("showroom.BeginSnipe:", t.name)
	return
}

// BeginSave callback
func (t Target) BeginSave() {
	log.Println("showroom.BeginSave:", t.name)
	return
}

// EndSave callback
func (t Target) EndSave(err error) {
	log.Println("showroom.EndSave:", t.name)
	return
}

// Name is the streamers real name
func (t Target) Name() string {
	return t.name
}

// Link is url string where this user's streams can be found
func (t Target) Link() string {
	return t.link
}

// Check gives nil if a stream has been found
func (t Target) Check() (streamURL string, err error) {
	// check to see if the user is live
	var isLive bool
	isLive, err = checkIsLive(t.id)
	if err == nil && !isLive {
		err = retry.StringError{
			Message: fmt.Sprintf("%s is not live yet", t.name),
			Attempt: t.Check,
		}

		return
	}

	// get the room for this user
	r, err := fetchRoom(t.link)
	if err != nil {
		err = retry.StringError{
			Message: fmt.Sprintf("showroom.Target.Check: %s %s", t.name, err),
			Attempt: t.Check,
		}

		return
	}

	// check for streaming url
	if r.StreamURL != "" {
		streamURL = r.StreamURL
		return
	}

	// check a new upcoming time
	nextLive := r.LiveRoom.NextLive
	if nextLive != "" && nextLive != "TBD" {
		at := parseUpcomingDate(nextLive)

		// there's a new date set so cancel this process and start a new one
		track.SnipeTargetAt(t, at)

		err = fmt.Errorf("%s has a new upcoming time set", t.name)

		return
	}

	err = retry.StringError{
		Message: fmt.Sprintf("%s has no stream yet", t.name),
		Attempt: t.Check,
	}

	return
}

// SavePath decides where videos are saved
func (t Target) SavePath() string {
	if t.name != "" {
		return t.name
	}

	return path.Base(t.link)
}
