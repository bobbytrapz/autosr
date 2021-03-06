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
	"strings"
	"sync"
	"time"

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

// GetDuration option
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
	defaultSavePath         = "autosr"
	configPathWindows       = `AppData\Roaming\autosr\`
	configPathUnix          = ".config/autosr/"
	defaultUserAgent        = `Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/71.0.3578.98 Safari/537.36`
	defaultStreamDownloader = `streamlink --http-header User-Agent={{UserAgent}} -o {{SavePath}} {{StreamURL}} best`
	defaultListenAddr       = ":4846"
	defaultPollRate         = 120 * time.Second
	defaultSelectFGColor    = "blue"
	defaultSelectBGColor    = "white"
)

// ConfigPath is the path where track list and config file are kept
var ConfigPath string

// EventHooks contains the names of valid event hooks
var EventHooks = []string{
	"begin-snipe",
	"begin-save",
	"end-save",
	"reload",
}

var v = viper.New()

func init() {
	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Println("options.init:", err)
		os.Exit(1)
	}

	// set defaults
	v.SetDefault("check_every", defaultPollRate)
	v.SetDefault("user_agent", defaultUserAgent)
	v.SetDefault("download_with", defaultStreamDownloader)
	v.SetDefault("listen_on", defaultListenAddr)
	v.SetDefault("select_fg_color", defaultSelectFGColor)
	v.SetDefault("select_bg_color", defaultSelectBGColor)

	v.SetConfigType(Format)
	v.SetConfigName(Filename)

	var configPath string
	if runtime.GOOS == "windows" {
		configPath = filepath.Join(home, configPathWindows)
	} else {
		configPath = filepath.Join(home, configPathUnix)
	}
	if err != nil {
		fmt.Println("options.init:", err)
		os.Exit(1)
	}

	savePath := filepath.Join(home, defaultSavePath)

	ConfigPath = configPath

	if err := os.MkdirAll(ConfigPath, 0700); err != nil {
		fmt.Println("error:", err)
		return
	}

	hooksPath := filepath.Join(ConfigPath, "hooks")

	if err := os.MkdirAll(hooksPath, 0700); err != nil {
		fmt.Println("error:", err)
		return
	}

	// make each hook directory
	for _, event := range EventHooks {
		p := filepath.Join(hooksPath, event)
		if err := os.MkdirAll(p, 0700); err != nil {
			fmt.Println("error:", err)
			return
		}
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
	})
}

var errInvalidPollRate = fmt.Errorf("error: time must be greater than 30s")

// AreValid is true if the options are valid
func AreValid() (ok bool, err error) {
	if v.GetDuration("check_every") < 30*time.Second {
		err = errInvalidPollRate
		return
	}

	downloader := v.GetString("download_with")
	sp := strings.Split(downloader, " ")
	app := sp[0]
	_, err = exec.LookPath(app)
	if err != nil {
		err = fmt.Errorf("error: could not find downloader: %s", err)
		return
	}

	return true, nil
}
