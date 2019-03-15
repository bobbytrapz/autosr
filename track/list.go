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

	f, err := os.OpenFile(listPath, os.O_CREATE|os.O_RDONLY, 0600)
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
	for _, t := range tracking {
		link := t.Link()
		if _, ok := lst[link]; !ok {
			if err := RemoveTarget(ctx, link); err != nil {
				fmt.Println(err)
				continue
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
		go func(l string) {
			defer waitAdd.Done()

			err := AddTarget(ctx, l)
			if err != nil {
				fmt.Println(err)
				return
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
		log.Println("track.readList: done")
	case <-ctx.Done():
		log.Println("track.readList:", ctx.Err())
	}

	return nil
}
