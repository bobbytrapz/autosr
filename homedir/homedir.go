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

package homedir

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
)

var cache string
var m sync.RWMutex

// Dir gives the user's home directory
func Dir() (dir string, err error) {
	m.RLock()
	c := cache
	m.RUnlock()

	if c != "" {
		return c, nil
	}

	m.Lock()
	defer m.Unlock()

	if runtime.GOOS == "windows" {
		dir, err = dirWindows()
	} else {
		dir, err = dirUnix()
	}

	if err == nil {
		cache = dir
	}

	return
}

// Expand tilde in a path
func Expand(path string) (string, error) {
	if path == "" || path[0] != '~' {
		return path, nil
	}

	if len(path) > 1 && path[1] != '/' && path[1] != '\\' {
		return "", fmt.Errorf("homedir.Expand: cannot expand path: '%s'", path)
	}

	dir, err := Dir()
	if err != nil {
		return "", err
	}

	return filepath.Join(dir, path[1:]), nil
}

func dirUnix() (string, error) {
	key := "HOME"
	if runtime.GOOS == "plan9" {
		key = "home"
	}

	if home := os.Getenv(key); home != "" {
		return home, nil
	}

	var stdout bytes.Buffer

	if runtime.GOOS == "darwin" {
		cmd := exec.Command("sh", "-c", `dscl -q . -read /Users/"$(whoami)" NFSHomeDirectory | sed 's/^[^ ]*: //'`)
		cmd.Stdout = &stdout
		if err := cmd.Run(); err == nil {
			result := strings.TrimSpace(stdout.String())
			if result != "" {
				return result, nil
			}
		}
	} else {
		// OS is not darwin
		cmd := exec.Command("getent", "passwd", strconv.Itoa(os.Getuid()))
		cmd.Stdout = &stdout
		err := cmd.Run()
		if err != nil && err != exec.ErrNotFound {
			return "", err
		}

		if out := strings.TrimSpace(stdout.String()); out != "" {
			// username:password:uid:gid:gecos:home:shell
			sp := strings.SplitN(out, ":", 7)
			if len(sp) > 5 {
				return sp[5], nil
			}
		}
	}

	stdout.Reset()
	cmd := exec.Command("sh", "-c", "cd && pwd")
	cmd.Stdout = &stdout
	if err := cmd.Run(); err != nil {
		return "", err
	}

	out := strings.TrimSpace(stdout.String())
	if out == "" {
		return "", errors.New("homedir.dirUnix: could not find home directory")
	}

	return out, nil
}

func dirWindows() (string, error) {
	if home := os.Getenv("HOME"); home != "" {
		return home, nil
	}

	if home := os.Getenv("USERPROFILE"); home != "" {
		return home, nil
	}

	drive := os.Getenv("HOMEDRIVE")
	path := os.Getenv("HOMEPATH")
	if drive == "" || path == "" {
		return "", errors.New("homeDir.dirWindows: could not find home directory")
	}

	home := drive + path

	return home, nil
}