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

package ipc

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/rpc"
	"os"
	"time"

	"github.com/bobbytrapz/autosr/options"
)

// Command to perform
type Command struct{}

var server *http.Server

// Start ipc server
func Start(ctx context.Context) {
	addr := options.Get("listen_on")

	c := &Command{}
	rpc.Register(c)
	rpc.HandleHTTP()

	server = &http.Server{
		Addr: addr,
	}

	// clean shutdown
	go func() {
		<-ctx.Done()
		log.Println("ipc: finishing...")
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		server.Shutdown(ctx)
		log.Println("ipc: done")
	}()

	go func() {
		log.Println("ipc.Start: ok")
		if err := server.ListenAndServe(); err != nil {
			if op, ok := err.(*net.OpError); ok {
				if op.Op == "listen" {
					// assume we failed to bind
					fmt.Println("autosr is already using port 4846")
					os.Exit(1)
				}
			}

			if err != http.ErrServerClosed {
				panic(err)
			}
		}
	}()
}
