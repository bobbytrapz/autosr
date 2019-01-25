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
	"bufio"
	"context"
	"fmt"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/bobbytrapz/autosr/options"
)

// list of urls to watch
var listPath = filepath.Join(options.ConfigPath, "track.list")

func readList(ctx context.Context) error {
	log.Println("track.readList: reading...")

	f, err := os.Open(listPath)
	if err != nil {
		return fmt.Errorf("track.readList: %s", err)
	}
	defer f.Close()

	// read valid urls from track list
	s := bufio.NewScanner(f)
	lst := make(map[string]bool, len(tracking))
	for s.Scan() {
		url := strings.TrimSpace(s.Text())
		if url == "" || url[0] == '#' {
			continue
		}
		lst[url] = true
	}

	// remove missing targets
	for host, t := range tracking {
		link := t.Link()
		if _, ok := lst[link]; !ok {
			m, err := FindModule(host)
			if err != nil {
				return err
			}
			ret, err := m.RemoveTarget(ctx, link)
			if err != nil {
				fmt.Println(host, ":", err)
				continue
			}
			if ret != nil {
				fmt.Println(host, ": removed", link)
			}
		}
	}

	// add targets
	var waitAdd sync.WaitGroup
	for link := range lst {
		select {
		case <-ctx.Done():
			log.Println("track.readList:", ctx.Err())
			break
		default:
		}
		waitAdd.Add(1)
		go func(u string) {
			defer waitAdd.Done()

			p, err := url.Parse(u)
			if err != nil {
				log.Println("track.readList: invalid url", u, err)
				return
			}
			host := p.Hostname()
			m, err := FindModule(host)
			if err != nil {
				log.Println("track.readList:", u, err)
				return
			}

			ret, err := m.AddTarget(ctx, u)
			if err != nil {
				fmt.Println(host, ":", err)
				return
			}
			if ret != nil {
				fmt.Println(host, ": added", u)
			}

			return
		}(link)
	}

	// wait until all urls have been added
	done := make(chan struct{}, 1)
	go func() {
		defer close(done)
		waitAdd.Wait()
	}()
	select {
	case <-done:
		log.Println("showroom.readTrackList: done")
	case <-ctx.Done():
		log.Println("showroom.readTrackList:", ctx.Err())
	}

	return nil
}
