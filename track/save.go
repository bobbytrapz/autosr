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
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/bobbytrapz/autosr/options"
)

// when a stream appears to end we wait to see if the user comes back
// online just in case there was a problem with the stream
var recoverTimeout = 5 * time.Minute

type saveTask struct {
	name string
	link string
}

var saving = struct {
	sync.RWMutex
	tasks map[saveTask]time.Time
}{
	tasks: make(map[saveTask]time.Time),
}

// find the most recent save for a link
func findSaveTask(link string) (task saveTask, createdAt time.Time) {
	saving.RLock()
	defer saving.RUnlock()

	// find the most recently added save task matching our target
	for t, at := range saving.tasks {
		if t.link == link {
			if createdAt.IsZero() || at.After(createdAt) {
				createdAt = at
				task = t
			}
		}
	}

	return
}

func hasSaveTask(task saveTask) bool {
	saving.RLock()
	defer saving.RUnlock()
	_, ok := saving.tasks[task]
	return ok
}

// give true if it is newly added
func addSaveTask(task saveTask) bool {
	if hasSaveTask(task) {
		return false
	}
	saving.Lock()
	defer saving.Unlock()
	saving.tasks[task] = time.Now()
	return true
}

func delSaveTask(task saveTask) {
	saving.Lock()
	defer saving.Unlock()
	delete(saving.tasks, task)
}

// record stream to disk using external program
func performSave(ctx context.Context, t *tracked, streamURL string) error {
	wg.Add(1)
	defer wg.Done()

	name := t.Name()

	link := t.Link()
	if link == "" {
		return errors.New("track.save: no link")
	}

	if streamURL == "" {
		return errors.New("track.save: no stream url")
	}

	task := saveTask{
		name: t.Name(),
		link: t.Link(),
	}
	if !addSaveTask(task) {
		log.Println("track.save: already saving", task.name)
		return nil
	}
	defer func() {
		delSaveTask(task)
		t.EndSave(ctx)
		runHooks("end-save", map[string]interface{}{
			"Name": task.name,
			"Link": task.link,
		})
	}()
	t.BeginSave(ctx)
	log.Println("track.save:", task.name)

	// used by command monitor to indicate that the command has exited
	exit := make(chan error, 1)

	// command information set in the closure below
	var cmd *exec.Cmd
	var app string
	var pid int
	var saveAs string

	// will be called again if we manage to recover a stream
	runSave := func(url string) error {
		var err error
		cmd, saveAs, err = runDownloader(ctx, url, name)
		if err != nil {
			return fmt.Errorf("track.save: %s", err)
		}

		if err := cmd.Start(); err != nil {
			return fmt.Errorf("track.save: %s", err)
		}
		app = cmd.Args[0]
		pid = cmd.Process.Pid
		log.Printf("track.save: %s [%s %d]", name, app, pid)
		runHooks("begin-save", map[string]interface{}{
			"Name":   task.name,
			"Link":   task.link,
			"SaveAs": saveAs,
		})

		// monitor downloader
		go func() {
			err := cmd.Wait()
			exit <- err
		}()

		return nil
	}

	// try to run the save command now
	if err := runSave(streamURL); err != nil {
		return err
	}

	// handle closing downloader
	for {
		select {
		case <-ctx.Done():
			cmd.Process.Kill()
			err := cmd.Wait()
			t.SetFinishedAt(time.Now())
			log.Printf("track.save: %s %s [%s %d] (%s)", name, ctx.Err(), app, pid, err)
			return nil
		case <-t.cancel:
			// we have been selected for cancellation
			cmd.Process.Kill()
			err := cmd.Wait()
			t.SetFinishedAt(time.Now())
			log.Printf("track.save: %s canceled [%s %d] (%s)", name, app, pid, err)
			return nil
		case err := <-exit:
			// something may have gone wrong so try to recover
			log.Printf("track.save: %s exited [%s %d]", name, app, pid)
			d, newURL, err := maybeRecover(ctx, t)
			if err != nil {
				// we did not recover so end this save
				t.SetFinishedAt(time.Now().Add(-d))
				return nil
			}
			log.Printf("track.save: %s recovered (%s)", name, d.Truncate(time.Millisecond))
			// run a new save command
			runSave(newURL)
		}
	}
}

type downloaderArgs struct {
	UserAgent string
	SavePath  string
	StreamURL string
}

// resembles go templates
var argRE = regexp.MustCompile("{{([^}]*)}}")

func (dargs downloaderArgs) ReplaceIn(command string) (app string, args []string) {
	sp := strings.Split(command, " ")
	var pats []string
	app, pats = sp[0], sp[1:]
	for _, pat := range pats {
		m := argRE.FindStringSubmatch(pat)
		if len(m) == 2 {
			// match found
			switch m[1] {
			case "UserAgent":
				arg := strings.Replace(pat, m[0], dargs.UserAgent, 1)
				args = append(args, arg)
			case "SavePath":
				arg := strings.Replace(pat, m[0], dargs.SavePath, 1)
				args = append(args, arg)
			case "StreamURL":
				arg := strings.Replace(pat, m[0], dargs.StreamURL, 1)
				args = append(args, arg)
			default:
				args = append(args, pat)
			}
		} else {
			// no match
			args = append(args, pat)
		}
	}

	return
}

// runs the user's downloader
func runDownloader(ctx context.Context, streamURL, name string) (cmd *exec.Cmd, saveAs string, err error) {
	// keep the path safe
	r := strings.NewReplacer(
		// linux
		".", "_",
		"/", "-",
		"\\", "-",
		"*", "★",
		// windows
		"<", "(",
		">", ")",
		":", "=",
		"\"", "-",
		"/", "-",
		"\\", "-",
		"|", "-",
		"?", "_",
		"*", "★",
	)
	name = r.Replace(name)

	saveTo := filepath.Join(options.Get("save_to"), name)
	ua := options.Get("user_agent")

	fn := fmt.Sprintf("%s-%s", time.Now().Format("2006-01-02"), name)
	saveAs = fn
	for n := 2; ; n++ {
		p := filepath.Join(saveTo, saveAs+".ts")
		if _, err := os.Stat(p); os.IsNotExist(err) {
			break
		}
		saveAs = fmt.Sprintf("%s %d", fn, n)
	}
	saveAs = filepath.Join(saveTo, saveAs+".ts")

	// replace placeholders
	command := options.Get("download_with")
	dargs := downloaderArgs{
		UserAgent: ua,
		SavePath:  saveAs,
		StreamURL: streamURL,
	}
	app, args := dargs.ReplaceIn(command)
	log.Printf("track.runDownloader: %s %s (%d)\n", app, args, len(args))
	cmd = exec.CommandContext(ctx, app, args...)

	err = os.MkdirAll(saveTo, os.ModePerm)
	if err != nil {
		err = fmt.Errorf("track.RunDownloader: %s", err)
		return
	}

	return cmd, saveAs, nil
}

func maybeRecover(ctx context.Context, t *tracked) (duration time.Duration, streamURL string, err error) {
	beginAt := time.Now()
	defer func() {
		endAt := time.Now()
		duration = endAt.Sub(beginAt)
	}()

	name := t.Name()
	log.Println("track.maybeRecover:", name, "recovering")

	err = waitForLive(ctx, t, recoverTimeout)
	if err != nil {
		log.Println("track.maybeRecover:", name, "is not online")
		err = errors.New("track.maybeRecover: target is not live")
		return
	}
	log.Println("track.maybeRecover:", name, "is online")

	streamURL, err = waitForStream(ctx, t, recoverTimeout)
	if err != nil {
		// we failed to find the new url
		log.Println("track.maybeRecover:", name, "did not find url")
		err = errors.New("track.maybeRecover: did not find url")
		return
	}

	log.Println("track.maybeRecover:", name, "found url")

	return
}
