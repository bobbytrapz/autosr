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

package cmd

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"runtime"

	"github.com/bobbytrapz/autosr/options"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(updateCmd)
}

const releasesURL = "https://api.github.com/repos/bobbytrapz/autosr/releases/latest"

func fail() {
	if isRunningInBackground() {
		fmt.Println("Please run 'autosr stop' before updating.")
		os.Exit(1)
	}

	fmt.Println("autosr update has failed.")
	fmt.Println("You may want to try reinstalling.")
	fmt.Println("https://github.com/bobbytrapz/autosr#readme")
	os.Exit(1)
}

func restoreFromBackup(backup, exePath string) {
	fmt.Println("Restoring from backup:", backup)
	err := os.Rename(backup, exePath)
	if err != nil {
		fmt.Println("Restore was not successful")
		fmt.Println("You may need to reinstall")
		fmt.Println("https://github.com/bobbytrapz/autosr#readme")
		os.Exit(1)
	}
}

// response from github api
type release struct {
	Name   string  `json:"name"`
	Assets []asset `json:"assets"`
}

type asset struct {
	Name        string `json:"name"`
	DownloadURL string `json:"browser_download_url"`
}

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Download and install the latest version",
	Run: func(cmd *cobra.Command, args []string) {
		if isRunningInBackground() {
			fmt.Println("Please run 'autosr stop' before updating.")
			os.Exit(1)
		}

		// prompt confirmation
		var p string
		fmt.Println("If the update fails you may need to reinstall.")
		fmt.Print("Proceed? [y/n] ")
		fmt.Scanln(&p)
		if p != "y" {
			return
		}

		var name string
		switch runtime.GOOS {
		case "darwin":
			name = "autosr-osx"
		case "windows":
			name = "autosr.exe"
		case "linux":
			name = "autosr"
		default:
			fmt.Println("autosr update is not supported for your OS.")
			os.Exit(1)
		}

		res, err := http.Get(releasesURL)
		if err != nil {
			fail()
		}

		buf, err := readResponse(res)
		if err != nil {
			fail()
		}
		res.Body.Close()

		var data release
		err = json.Unmarshal(buf.Bytes(), &data)
		if err != nil {
			fail()
		}

		var dl string
		for _, a := range data.Assets {
			if a.Name == name {
				dl = a.DownloadURL
			}
		}
		fmt.Println("Installing", data.Name)

		if err != nil {
			fmt.Println(err)
			fail()
		}

		fmt.Println("Download from", dl)
		res, err = http.Get(dl)
		if err != nil {
			fail()
		}

		autosr, err := readResponse(res)
		if err != nil {
			fail()
		}
		res.Body.Close()

		exePath, err := os.Executable()
		if err != nil {
			fail()
		}

		backup := filepath.Join(options.ConfigPath, "autosr.backup")
		fmt.Println("Backup current version to", backup)
		err = os.Rename(exePath, backup)
		if err != nil {
			if os.IsPermission(err) {
				fmt.Println("Permission was denied")
				if runtime.GOOS != "windows" {
					fmt.Println("You could try 'sudo autosr update'")
				}
			}
			fail()
		}

		fmt.Println("Writing new version to", exePath)
		err = ioutil.WriteFile(exePath, autosr.Bytes(), 0775)
		if err != nil {
			if os.IsPermission(err) {
				fmt.Println("Permission was denied")
				if runtime.GOOS != "windows" {
					fmt.Println("You could try 'sudo autosr update'")
				}
			}
			restoreFromBackup(backup, exePath)
			fail()
		}

		fmt.Println("Install ok.")
	},
}

func readResponse(res *http.Response) (buf *bytes.Buffer, err error) {
	encoding := res.Header.Get("Content-Encoding")
	var r io.ReadCloser
	switch encoding {
	case "gzip":
		r, err = gzip.NewReader(res.Body)
		defer r.Close()
	default:
		r = res.Body
	}

	if err != nil {
		err = fmt.Errorf("update.readReponse: %s", err)
		return
	}

	buf = &bytes.Buffer{}
	io.Copy(buf, r)

	return
}
