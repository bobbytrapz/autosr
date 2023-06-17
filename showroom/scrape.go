package showroom

import (
	"context"
	"errors"
	"fmt"
	"log"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
)

var (
	launchBrowserOnce sync.Once

	browser *rod.Browser

	pagePool rod.PagePool
)

const (
	MaxNumberPages = 25
)

func scrapeRoomData(ctx context.Context, link string) (roomStatus, error) {
	launchBrowserOnce.Do(func() {
		l := launcher.New()
		u := l.MustLaunch()
		browser = rod.New().
			ControlURL(u).
			MustConnect()
		pagePool = rod.NewPagePool(MaxNumberPages)
		go func() {
			for {
				select {
				case <-ctx.Done():
					log.Print("showroom.scrapeRoomData: closing browser...")
					l.Cleanup()
					browser.MustClose()
					return
				}
			}
		}()
	})

	var status roomStatus

	page := pagePool.Get(func() *rod.Page {
		return browser.MustPage()
	})
	defer pagePool.Put(page)
	page.Navigate(link)

	check := func(err error) {
		var evalErr *rod.ErrEval
		if errors.Is(err, context.DeadlineExceeded) { // timeout error
			log.Println("showroom.scrapeRoomData: page timeout:", link)
		} else if errors.As(err, &evalErr) { // eval error
			log.Println("showroom.scrapeRoomData: error:", link+":", evalErr)
		} else if err != nil {
			log.Println("showroom.scrapeRoomData:", link+":", err)
		}
	}

	expectedURLKey := path.Base(link)
	err := rod.Try(func() {
		waitFor := 10 * time.Second

		jsCheckFn := func(key string) string {
			// format := `() => !!(Object.keys((window['__NUXT__'] && window['__NUXT__']['data']) || {}).map((k) => (window['__NUXT__']['data'] || {})[k] || {}).filter(o => !!o['%s']).pop() || {})['%s']`
			format := `() => !!(Object.keys((window['__NUXT__'] && window['__NUXT__']['data']) || {}).filter(k => k.includes('%s')).map(k => (window['__NUXT__']['data'] || {})[k] || {}).filter(o => !!o['%s']).pop() || {})['%s']`
			return fmt.Sprintf(format, expectedURLKey, key, key)
		}

		jsGetFn := func(key string) string {
			// format := `() => (Object.keys((window['__NUXT__'] && window['__NUXT__']['data']) || {}).map((k) => (window['__NUXT__']['data'] || {})[k] || {}).filter(o => !!o['%s']).pop() || {})['%s']`
			format := `() => (Object.keys((window['__NUXT__'] && window['__NUXT__']['data']) || {}).filter(k => k.includes('%s')).map(k => (window['__NUXT__']['data'] || {})[k] || {}).filter(o => !!o['%s']).pop() || {})['%s']`
			return fmt.Sprintf(format, expectedURLKey, key, key)
		}

		roomID := page.Timeout(waitFor).
			MustWait(jsCheckFn("room_id")).
			MustEval(jsGetFn("room_id"))

		roomURLKey := page.Timeout(waitFor).
			MustWait(jsCheckFn("room_url_key")).
			MustEval(jsGetFn("room_url_key"))

		roomName := page.Timeout(waitFor).
			MustWait(jsCheckFn("room_name")).
			MustEval(jsGetFn("room_name"))

		log.Print("showroom.scrapeRoomData: room_id:", roomID)
		log.Print("showroom.scrapeRoomData: room_url_key:", roomURLKey)
		log.Print("showroom.scrapeRoomData: room_name:", roomName)

		status.ID = roomID.Int()
		status.LiveRoom.URLKey = roomURLKey.String()
		status.Name = roomName.String()
	})
	check(err)

	// confirm room url key
	if strings.ToLower(status.LiveRoom.URLKey) != strings.ToLower(expectedURLKey) {
		return roomStatus{}, fmt.Errorf("unexpected room url key")
	}

	return status, err
}
