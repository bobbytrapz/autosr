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
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"time"

	"github.com/bobbytrapz/autosr/options"
	"github.com/bobbytrapz/autosr/retry"
	"golang.org/x/net/html"
	"golang.org/x/net/publicsuffix"
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
	req.Header.Add("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Add("Accept-Encoding", "gzip, deflate, br")
	req.Header.Add("Accept-Language", "en-US,en;q=0.5")
	req.Header.Add("Cache-Control", "no-cache")
	req.Header.Add("Connection", "keep-alive")
	req.Header.Add("DNT", "1")
	req.Header.Add("Host", domainName)
	req.Header.Add("Pragma", "no-cache")
	req.Header.Add("Upgrade-Insecure-Requests", "1")

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
	req, err = makeRequest(ctx, "GET", url, nil, onlivesURL)
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

func makeJSONRequest(ctx context.Context, endpoint string, id int) (req *http.Request, err error) {
	url := fmt.Sprintf("%s?room_id=%d", endpoint, id)
	req, err = makeRequest(ctx, "GET", url, nil, "")
	if err != nil {
		return
	}

	// xhr headers but we decided to copy the browser headers closely instead
	// req.Header.Add("Accept", "application/json")
	// req.Header.Add("X-Requested-With", "XMLHttpRequest")

	return
}

func makeIsLiveRequest(ctx context.Context, id int) (req *http.Request, err error) {
	return makeJSONRequest(ctx, "https://www.showroom-live.com/room/is_live", id)
}

func makeStreamingURLRequest(ctx context.Context, id int) (req *http.Request, err error) {
	return makeJSONRequest(ctx, "https://www.showroom-live.com/api/live/streaming_url", id)
}

func makeNextLiveRequest(ctx context.Context, id int) (req *http.Request, err error) {
	return makeJSONRequest(ctx, "https://www.showroom-live.com/api/room/next_live", id)
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

func checkStreamURL(ctx context.Context, id int) (streamURL string, err error) {
	req, err := makeStreamingURLRequest(ctx, id)
	if err != nil {
		return
	}

	res, err := httpClient.Do(req)
	if err != nil {
		err = retry.StringError{
			Message: fmt.Sprintf("showroom.checkStreamURL: %s", err),
			Attempt: func() (string, error) {
				return checkStreamURL(ctx, id)
			},
		}

		return
	}
	defer res.Body.Close()

	buf, err := readResponse(res)
	if err != nil {
		err = retry.StringError{
			Message: fmt.Sprintf("showroom.checkStreamURL: %s", err),
			Attempt: func() (string, error) {
				return checkStreamURL(ctx, id)
			},
		}

		return
	}

	var data streamingURLResponse
	if err = json.Unmarshal(buf.Bytes(), &data); err != nil {
		err = retry.StringError{
			Message: fmt.Sprintf("showroom.checkStreamURL: %s", err),
			Attempt: func() (string, error) {
				return checkStreamURL(ctx, id)
			},
		}

		return
	}

	// find highest quality hls stream
	var stream stream
	for _, s := range data.StreamingURLs {
		if s.Type != "hls" {
			continue
		}

		if s.Quality > stream.Quality {
			stream = s
		}
	}
	streamURL = stream.URL

	return
}

func checkNextLive(ctx context.Context, id int) (at time.Time, err error) {
	req, err := makeNextLiveRequest(ctx, id)
	if err != nil {
		return
	}

	res, err := httpClient.Do(req)
	if err != nil {
		err = retry.TimeError{
			Message: fmt.Sprintf("showroom.checkNextLive: %s", err),
			Attempt: func() (time.Time, error) {
				return checkNextLive(ctx, id)
			},
		}

		return
	}
	defer res.Body.Close()

	buf, err := readResponse(res)
	if err != nil {
		err = retry.TimeError{
			Message: fmt.Sprintf("showroom.checkNextLive: %s", err),
			Attempt: func() (time.Time, error) {
				return checkNextLive(ctx, id)
			},
		}

		return
	}

	var data nextLiveResponse
	if err = json.Unmarshal(buf.Bytes(), &data); err != nil {
		err = retry.TimeError{
			Message: fmt.Sprintf("showroom.checkNextLive: %s", err),
			Attempt: func() (time.Time, error) {
				return checkNextLive(ctx, id)
			},
		}

		return
	}

	if data.Epoch > 0 {
		at = time.Unix(data.Epoch, 0).Local()
	}

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

func fetchPage(ctx context.Context, link string) (*html.Node, error) {
	req, err := makeRequest(ctx, "GET", link, nil, "")
	if err != nil {
		return nil, err
	}
	res, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	buf, err := readResponse(res)
	if err != nil {
		err = fmt.Errorf("readResponse %s", err)
		return nil, err
	}

	doc, err := html.Parse(buf)
	if err != nil {
		return nil, err
	}

	return doc, err
}

// information about a user's room is parsed from their page
func fetchRoom(ctx context.Context, link string) (roomStatus, error) {
	return scrapeRoomData(ctx, link)
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

	if res.StatusCode != 200 {
		err = fmt.Errorf("showroom.readReponse: %s", res.Status)
		return
	}

	buf = &bytes.Buffer{}
	io.Copy(buf, r)

	return
}
