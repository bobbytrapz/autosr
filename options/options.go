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

package options

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	"github.com/bobbytrapz/autosr/homedir"
	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
)

var m sync.RWMutex

// Get an option
func Get(k string) string {
	m.RLock()
	defer m.RUnlock()

	return v.GetString(k)
}

// GetDuration an option
func GetDuration(k string) time.Duration {
	m.RLock()
	defer m.RUnlock()

	return v.GetDuration(k)
}

const (
	// Filename for config file
	Filename = "autosr"
	// Format for config file
	Format                  = "toml"
	defaultSavePath         = "~/autosr"
	configPathWindows       = `~\AppData\Roaming\autosr\`
	configPathUnix          = "~/.config/autosr/"
	defaultUserAgent        = `Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/71.0.3578.98 Safari/537.36`
	defaultStreamDownloader = "streamlink"
	defaultListenAddr       = "0.0.0.0:4846"
	defaultPollRate         = 120 * time.Second
	defaultSelectFGColor    = "black"
	defaultSelectBGColor    = "white"
)

// ConfigPath is the path where track list and config file are kept
var ConfigPath string

var v = viper.New()

func init() {
	// set defaults
	v.SetDefault("check_every", defaultPollRate)
	v.SetDefault("user_agent", defaultUserAgent)
	v.SetDefault("download_with", defaultStreamDownloader)
	v.SetDefault("listen_on", defaultListenAddr)
	v.SetDefault("select_fg_color", defaultSelectFGColor)
	v.SetDefault("select_bg_color", defaultSelectBGColor)

	v.SetConfigType(Format)
	v.SetConfigName(Filename)

	var err error
	var configPath string
	if runtime.GOOS == "windows" {
		configPath, err = homedir.Expand(configPathWindows)
	} else {
		configPath, err = homedir.Expand(configPathUnix)
	}
	if err != nil {
		fmt.Println("options.init:", err)
		os.Exit(1)
	}

	savePath, err := homedir.Expand(defaultSavePath)
	if err != nil {
		fmt.Println("options.init:", err)
		os.Exit(1)
	}

	ConfigPath = configPath

	if err := os.MkdirAll(ConfigPath, 0700); err != nil {
		fmt.Println("error:", err)
		return
	}

	v.SetDefault("save_to", savePath)
	v.AddConfigPath(configPath)

	if err := v.ReadInConfig(); err != nil {
		p := filepath.Join(configPath, Filename+"."+Format)
		if err := v.WriteConfigAs(p); err != nil {
			panic(err)
		}
		fmt.Println("[ok] wrote new config file")
	}

	v.WatchConfig()
	v.OnConfigChange(func(e fsnotify.Event) {
		if ok, err := AreValid(); !ok {
			if err == errInvalidPollRate {
				v.Set("check_every", 1*time.Minute)
			}
		}
		fmt.Println("config file changed:", e.Name)
	})
}

var errInvalidPollRate = fmt.Errorf("error: time must be greater than 30s")

// AreValid is true if the options are valid
func AreValid() (ok bool, err error) {
	if v.GetDuration("check_every") < 30*time.Second {
		err = errInvalidPollRate
		return
	}

	_, err = exec.LookPath(v.GetString("download_with"))
	if err != nil {
		err = fmt.Errorf("error: could not find downloader: %s", err)
		return
	}

	return true, nil
}
