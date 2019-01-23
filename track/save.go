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
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"github.com/bobbytrapz/autosr/options"
)

var saving = struct {
	sync.RWMutex
	lookup map[string]*exec.Cmd
}{
	lookup: make(map[string]*exec.Cmd),
}

func hasSave(link string) bool {
	saving.RLock()
	defer saving.RUnlock()
	_, ok := saving.lookup[link]
	return ok
}

func addSave(link string, cmd *exec.Cmd) error {
	saving.Lock()
	defer saving.Unlock()
	if _, ok := saving.lookup[link]; ok {
		return errors.New("track.addSave: stream is already being downloaded")
	}
	saving.lookup[link] = cmd
	return nil
}

func delSave(link string) {
	saving.Lock()
	defer saving.Unlock()
	delete(saving.lookup, link)
}

// Save recording of a stream to disk
func Save(ctx context.Context, tracked *tracked) error {
	name := tracked.Name()

	link := tracked.Link()
	if link == "" {
		return errors.New("track.Save: no link")
	}

	streamURL := tracked.StreamURL()
	if streamURL == "" {
		return errors.New("track.Save: no stream url")
	}

	tracked.SetStartedAt(time.Now())
	tracked.BeginSave()

	cmd, err := RunDownloader(ctx, streamURL, name)
	if err != nil {
		return fmt.Errorf("track.Save: %s", err)
	}

	if err := addSave(link, cmd); err != nil {
		return fmt.Errorf("track.Save: %s", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("track.Save: %s", err)
	}
	app := cmd.Args[0]
	pid := cmd.Process.Pid
	log.Printf("track.Save: %s [%s %d]", name, app, pid)

	cancelSave := make(chan struct{})
	tracked.SetCancel(cancelSave)

	exit := make(chan struct{}, 1)
	// monitor downloader
	go func() {
		defer close(exit)
		cmd.Wait()
	}()

	// handle closing downloader
	wg.Add(1)
	go func() {
		defer wg.Done()

		for {
			select {
			case <-ctx.Done():
				delSave(link)
				cmd.Process.Kill()
				err := cmd.Wait()
				log.Printf("track.Save: %s %s [%s %d] (%s)", name, ctx.Err(), app, pid, err)
				tracked.EndSave(nil)
				tracked.SetFinishedAt(time.Now())
				return
			case <-cancelSave:
				// we have been selected for cancellation
				delSave(link)
				cmd.Process.Kill()
				err := cmd.Wait()
				log.Printf("track.Save: %s canceled [%s %d] (%s)", name, app, pid, err)
				tracked.EndSave(nil)
				tracked.SetFinishedAt(time.Now())
				return
			case <-exit:
				if hasSave(link) {
					delSave(link)
					// something may have gone wrong so try again right now
					log.Printf("track.Save: %s exited [%s %d] (%s)", name, app, pid, err)
					snipeEnded(ctx, tracked, time.Now())
				} else {
					log.Printf("track.Save: %s done [%s %d] (%s)", name, app, pid, err)
				}
				return
			}
		}
	}()

	return nil
}

// RunDownloader runs the user's downloader
func RunDownloader(ctx context.Context, url, name string) (cmd *exec.Cmd, err error) {
	saveTo := filepath.Join(options.Get("save_to"), name)
	ua := fmt.Sprintf("User-Agent=%s", options.Get("user_agent"))
	app := options.Get("download_with")

	fn := fmt.Sprintf("%s-%s", time.Now().Format("2006-01-02"), name)
	saveAs := fn
	for n := 1; ; n++ {
		p := filepath.Join(saveTo, saveAs+".ts")
		if _, err := os.Stat(p); os.IsNotExist(err) {
			break
		}
		saveAs = fmt.Sprintf("%s %d", fn, n)
	}
	saveAs = saveAs + ".ts"

	args := []string{
		"--hls-segment-threads", "4",
		"--hls-segment-timeout", "2.0",
		"--http-timeout", "2.0",
		"--http-header", ua,
		"-o", saveAs,
		fmt.Sprintf("hlsvariant://%s", url),
		"best",
	}

	cmd = exec.CommandContext(ctx, app, args...)
	err = os.MkdirAll(saveTo, os.ModePerm)
	if err != nil {
		err = fmt.Errorf("track.RunDownloader: %s", err)
		return
	}
	cmd.Dir = saveTo
	setArgs(cmd)

	return cmd, nil
}
