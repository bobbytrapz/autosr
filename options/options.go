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
)

// ConfigPath is the path where track list and config file are kept
var ConfigPath string

var v = viper.New()

func init() {
	// set defaults
	v.SetDefault("poll_rate", defaultPollRate)
	v.SetDefault("user_agent", defaultUserAgent)
	v.SetDefault("stream_downloader", defaultStreamDownloader)
	v.SetDefault("listen_addr", defaultListenAddr)

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

	v.SetDefault("SavePath", savePath)
	v.AddConfigPath(configPath)

	if err := v.ReadInConfig(); err != nil {
		p := filepath.Join(configPath, Filename)
		if err := v.WriteConfigAs(p); err != nil {
			panic(err)
		}
		fmt.Println("[ok] wrote new config file")
	}

	v.WatchConfig()
	v.OnConfigChange(func(e fsnotify.Event) {
		fmt.Println("config file changed:", e.Name)
	})
}
