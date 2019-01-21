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
	"time"

	"github.com/bobbytrapz/autosr/options"
)

// Save recording of a stream to disk
func Save(ctx context.Context, tracked *tracked) error {
	if tracked.Status() == saving {
		return errors.New("track.Saving: already saving")
	}

	url := tracked.StreamURL()
	if url == "" {
		return errors.New("track.Save: no stream url")
	}

	tracked.SetStatus(saving)
	tracked.SetStartedAt(time.Now())
	tracked.Target.BeginSave()

	cmd, err := RunDownloader(ctx, url, tracked.Target.Name())
	if err != nil {
		return fmt.Errorf("track.Save: %s", err)
	}
	exit := make(chan struct{}, 1)

	// handle canceling downloader
	go func() {
		tracked.Lock()
		defer tracked.Unlock()

		wg.Add(1)
		defer wg.Done()

		for {
			select {
			case <-ctx.Done():
				log.Println("track.Save:", tracked.Target.Name(), "canceled")
			case <-exit:
				tracked.SetStatus(sleeping)
				tracked.SetFinishedAt(time.Now())
				tracked.Target.EndSave(nil)
				return
			}
		}
	}()

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("track.Save: %s", err)
	}
	log.Printf("track.Save: %s [%s %d]", tracked.Target.Name(), cmd.Args[0], cmd.Process.Pid)

	// monitor downloader
	go func() {
		if err := cmd.Wait(); err != nil {
			log.Println("track.Save:", tracked.Target.Name(), err)
			// something went wrong so try again right now
			SnipeTargetAt(tracked.Target, time.Now())
		}
		exit <- struct{}{}
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
		saveAs = fmt.Sprintf("%s-%d", fn, n)
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

	return
}
