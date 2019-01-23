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

package showroom

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/bobbytrapz/autosr/retry"
	"golang.org/x/net/html"
	"golang.org/x/net/publicsuffix"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"time"

	"github.com/bobbytrapz/autosr/options"
)

const (
	domainName = "www.showroom-live.com"
	onlivesURL = "https://www.showroom-live.com/onlive"
)

var httpCookieJar *cookiejar.Jar
var httpClient http.Client

func init() {
	var err error
	httpCookieJar, err = cookiejar.New(&cookiejar.Options{PublicSuffixList: publicsuffix.List})
	if err != nil {
		panic(err)
	}
	httpClient = http.Client{
		Jar:     httpCookieJar,
		Timeout: 60 * time.Second,
	}
}

func makeRequest(ctx context.Context, method, url string, body io.Reader, referer string) (req *http.Request, err error) {
	ua := options.Get("user_agent")

	req, err = http.NewRequest(method, url, body)
	if err != nil {
		return
	}

	// headers
	req.Header.Add("Host", domainName)
	req.Header.Add("Connection", "keep-alive")
	req.Header.Add("Accept-Encoding", "gzip, deflate, sdch, br")
	req.Header.Add("Accept-Language", "en-US,en,q=0.8")
	if referer != "" {
		req.Header.Add("Referer", referer)
	}
	req.Header.Add("User-Agent", ua)

	req = req.WithContext(ctx)

	return
}

func onlivesAPI() string {
	return fmt.Sprintf("https://www.showroom-live.com/api/live/onlives?_=%d", time.Now().Unix())
}

func makeOnLivesRequest(ctx context.Context) (req *http.Request, err error) {
	url := fmt.Sprintf("https://www.showroom-live.com/api/live/onlives?_=%d", time.Now().Unix())
	req, err = makeRequest(ctx, "get", url, nil, onlivesURL)
	if err != nil {
		return
	}

	// note: this cookie means we only get results for Idols genre
	c := &http.Cookie{Name: "genreTabOnLive", Value: "102"}
	req.AddCookie(c)

	// xhr headers
	req.Header.Add("Accept", "application/json")
	req.Header.Add("X-Requested-With", "XMLHttpRequest")
	return req, nil
}

func makeIsLiveRequest(ctx context.Context, id int) (req *http.Request, err error) {
	url := fmt.Sprintf("https://www.showroom-live.com/room/is_live?room_id=%d", id)
	req, err = makeRequest(ctx, "get", url, nil, domainName)
	if err != nil {
		return
	}

	// xhr headers
	req.Header.Add("Accept", "application/json")
	req.Header.Add("X-Requested-With", "XMLHttpRequest")

	return
}

// tells us if a certain showroom user is online
func checkIsLive(ctx context.Context, id int) (isLive bool, err error) {
	req, err := makeIsLiveRequest(ctx, id)
	if err != nil {
		return
	}

	res, err := httpClient.Do(req)
	if err != nil {
		err = retry.BoolError{
			Message: fmt.Sprintf("showroom.checkIsLive: %s", err),
			Attempt: func() (bool, error) {
				return checkIsLive(ctx, id)
			},
		}

		return
	}
	defer res.Body.Close()

	buf, err := readResponse(res)
	if err != nil {
		err = retry.BoolError{
			Message: fmt.Sprintf("showroom.checkIsLive: %s", err),
			Attempt: func() (bool, error) {
				return checkIsLive(ctx, id)
			},
		}
		return
	}

	var data isLiveResponse
	if err = json.Unmarshal(buf.Bytes(), &data); err != nil {
		err = retry.BoolError{
			Message: fmt.Sprintf("showroom.checkIsLive: %s", err),
			Attempt: func() (bool, error) {
				return checkIsLive(ctx, id)
			},
		}

		return
	}

	isLive = (data.Ok == 1)

	return
}

// fetch all showrooms
func fetchAllRooms(ctx context.Context) (rooms []room, err error) {
	req, err := makeOnLivesRequest(ctx)
	if err != nil {
		err = fmt.Errorf("showroom.fetchRooms: %s", err)
		return
	}

	res, err := httpClient.Do(req)
	if err != nil {
		return
	}
	defer res.Body.Close()

	buf, err := readResponse(res)
	if err != nil {
		err = fmt.Errorf("showroom.fetchRooms: %s", err)
		return
	}

	var data onlivesResponse
	if err = json.Unmarshal(buf.Bytes(), &data); err != nil {
		err = fmt.Errorf("showroom.fetchRooms: %s", err)
		return
	}

	// note: this works because a cookie is set for Idol genre only
	if len(data.Onlives) > 0 {
		rooms = data.Onlives[0].Rooms
	}

	return
}

// information about a user's room is parsed from thier page
func fetchRoom(ctx context.Context, link string) (status roomStatus, err error) {
	req, err := makeRequest(ctx, "get", link, nil, "")
	if err != nil {
		return
	}
	res, err := httpClient.Do(req)
	if err != nil {
		return
	}
	defer res.Body.Close()

	buf, err := readResponse(res)
	if err != nil {
		err = fmt.Errorf("readResponse %s", err)
		return
	}

	doc, err := html.Parse(buf)
	if err != nil {
		return
	}

	status, err = findInitialData(doc)

	return
}

// CommentURLFromRoomID given an id gives the url comments can be fetched from
func CommentURLFromRoomID(ctx context.Context, id int) url.URL {
	return url.URL{
		Scheme: "https",
		Host:   "www.showroom-live.com",
		Path:   fmt.Sprintf("/api/live/comment_log?room_id=%d", id),
	}
}

// FetchComments from url
func FetchComments(ctx context.Context, id int) []Comment {
	return nil
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
		err = fmt.Errorf("showroom.readReponse: %s", err)
		return
	}

	buf = &bytes.Buffer{}
	io.Copy(buf, r)

	return
}

// find the js-initial-data and js-live-data tags and unmarshal them
func findInitialData(doc *html.Node) (status roomStatus, err error) {
	var data string
	var liveData string
	var fn func(*html.Node)
	fn = func(n *html.Node) {
		if n.Type == html.ElementNode {
			switch n.Data {
			case "script":
				for _, a := range n.Attr {
					if a.Key == "id" {
						switch a.Val {
						case "js-initial-data":
							for _, a := range n.Attr {
								if a.Key == "data-json" {
									data = a.Val
									return
								}
							}
						case "js-live-data":
							for _, a := range n.Attr {
								if a.Key == "data-json" {
									liveData = a.Val
									return
								}
							}
						}
					}
				}
			}
		}

		for c := n.FirstChild; c != nil; c = c.NextSibling {
			fn(c)
		}
	}
	fn(doc)

	if data != "" {
		buf := bytes.NewBufferString(data)
		err = json.Unmarshal(buf.Bytes(), &status)
		if err != nil {
			err = fmt.Errorf("showroom.findInitialData: %s", err)
			return
		}
	}

	if liveData != "" {
		buf := bytes.NewBufferString(liveData)
		err = json.Unmarshal(buf.Bytes(), &status)
		if err != nil {
			err = fmt.Errorf("showroom.findInitialData: %s", err)
			return
		}
	}

	if data == "" && liveData == "" {
		err = errors.New("showroom.findInitialData: no initial data")
		return
	}

	return
}

func parseUpcomingDate(d string) time.Time {
	layout := "1/2 15:04 2006"
	now := time.Now()
	date := fmt.Sprintf("%s %s", d, now.Format(("2006")))

	at, err := time.Parse(layout, date)
	if err != nil {
		// assume now if the date is formatted wrong for whatever reason
		return now
	}

	if at.Month() < now.Month() {
		at = at.AddDate(1, 0, 0)
	}

	// FIXME: time is in JST but UTC so we do this crap
	at = at.Add(-14*time.Hour + 5*time.Hour).Local()

	return at
}
