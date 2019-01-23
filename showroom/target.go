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
	"bytes"
	"fmt"
	"log"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/bobbytrapz/autosr/retry"
	"github.com/bobbytrapz/autosr/track"
)

// note: this is arbitrary
const maxDisplayLength = 75

// AddTargetFromURL adds showroom user using the url
// returns true if they were actually added
func AddTargetFromURL(link string) (bool, error) {
	_, err := url.Parse(link)
	if err != nil {
		return false, fmt.Errorf("showroom.AddTargetFromURL: '%s' %s", link, err)
	}

	s, err := fetchRoom(link)
	if err != nil {
		return false, fmt.Errorf("showroom.AddTargetFromURL: '%s' %s", link, err)
	}

	name := strings.TrimSpace(s.Name)
	var buf bytes.Buffer
	for _, r := range name {
		buf.WriteRune(r)
		if len(buf.String()) > maxDisplayLength {
			break
		}
		if r != ' ' && r != '(' && r != ')' {
			buf.WriteRune(' ')
		}
	}

	t := Target{
		name:    name,
		display: buf.String(),
		id:      s.ID,
		link:    link,
		urlKey:  s.LiveRoom.URLKey,
	}

	err = track.AddTarget(t)
	if err != nil {
		// we must be already targeting
		return false, nil
	}

	m.Lock()
	targets = append(targets, t)
	m.Unlock()

	// check target right away
	if streamURL, err := t.checkRoom(); err == nil {
		log.Println("showroom.AddTargetFromURL:", t.name, "is live now!", streamURL)
		// they are live now so snipe them now
		if err = track.SnipeTargetAt(t, time.Now()); err != nil {
			log.Println("showroom.AddTargetFromURL:", err)
			return false, nil
		}
		return true, nil
	}

	return true, nil
}

// RemoveTargetFromURL removes showroom user using the url
// returns true if they were actually removed
func RemoveTargetFromURL(link string) (bool, error) {
	if err := track.RemoveTarget(link); err != nil {
		// this target does not exist
		return false, nil
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

	return true, nil
}

// Target showroom streamer
type Target struct {
	// info
	name    string
	display string
	id      int
	link    string
	urlKey  string
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

// Display for display in dashboard
func (t Target) Display() string {
	return t.display
}

// Name is the streamers real name
func (t Target) Name() string {
	return t.name
}

// Link is url string where this user's streams can be found
func (t Target) Link() string {
	return t.link
}

// Check gives nil if a stream has been found expects user to possibly be live
func (t Target) Check() (streamURL string, err error) {
	// check to see if the user is live
	// if not just give up now
	var isLive bool
	isLive, err = checkIsLive(t.id)
	if err == nil && !isLive {
		err = retry.StringError{
			Message: fmt.Sprintf("%s is not live yet", t.name),
			Attempt: t.Check,
		}

		return
	}

	// they are live so check their room
	return t.checkRoom()
}

// check the user's actual room page
// they may not actually be live but we can get the upcoming time as well
func (t Target) checkRoom() (streamURL string, err error) {
	wg.Add(1)
	defer wg.Done()
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

		// there's a new date set so start a new one
		if err = track.SnipeTargetAt(t, at); err != nil {
			log.Println("showroom.Check:", err)

			return
		}

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
