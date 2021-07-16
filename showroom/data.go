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
	"net/url"
	"strings"
	"time"
)

type onlivesResponse struct {
	Host    string      `json:"bcsvr_host"`
	Port    int         `json:"bcsvr_port"`
	Onlives []livesData `json:"onlives"`
}

type isLiveResponse struct {
	Ok int `json:"ok"`
}

type streamingURLResponse struct {
	StreamingURLs []stream `json:"streaming_url_list"`
}

type nextLiveResponse struct {
	Epoch int64  `json:"epoch"`
	Text  string `json:"text"`
}

type livesData struct {
	Genre       int    `json:"genre_id"`
	HasUpcoming bool   `json:"has_upcoming"`
	GenreName   string `json:"genre_name"`
	Rooms       []room `json:"lives"`
}

type room struct {
	Key           string   `json:"bcsvr_key"`
	CellType      int      `json:"cell_type"`
	Color         string   `json:"color"`
	FollowerNum   int      `json:"follower_num"`
	GenreID       int      `json:"genre_id"`
	Image         string   `json:"image"`
	ImageLive     string   `json:"image_live"`
	IsFollow      bool     `json:"is_follow"`
	LiveID        int      `json:"live_id"`
	LiveType      int      `json:"live_type"`
	MainName      string   `json:"main_name"`
	OfficialLv    int      `json:"official_lv"`
	ID            int      `json:"room_id"`
	URLKey        string   `json:"room_url_key"`
	StartedAt     int64    `json:"started_at"`
	StreamingURLs []stream `json:"streaming_url_list"`
	Telop         string   `json:"telop"`
	ViewNum       int      `json:"view_num"`
	// internal
	lastStatusAt time.Time
	lastStatus   roomStatus
}

func (r *room) link() string {
	u := url.URL{
		Scheme: "https",
		Host:   domainName,
		Path:   r.URLKey,
	}
	return u.String()
}

// data about a room fetched from the user page
type roomStatus struct {
	// js-initial-data
	StartedAt  int64  `json:"startedAt"`
	Live       bool   `json:"isLive"`
	StreamURL  string `json:"streamingUrlHls"`
	AnteroomID int    `json:"anteroomId"`
	ID         int    `json:"roomId"`
	LiveID     int    `json:"liveId"`
	Name       string `json:"roomName"`
	// LiveStatus int    `json:"liveStatus"`
	// js-live-data
	// "month/day hour:minute"
	// ID       string       `json:"room_id"`
	LiveRoom liveRoomInfo `json:"room"`
}

// room section of roomStatus
type liveRoomInfo struct {
	URLKey      string `json:"room_url_key"`
	LastLive    string `json:"last_lived_at"`
	NextLive    string `json:"next_live"`
	YouTubeID   string `json:"youtube_id"`
	FollowerNum int `json:"follower_num"`
}

// a showroom stream
type stream struct {
	ID      int    `json:"id"`
	Default bool   `json:"is_default"`
	Label   string `json:"label"`
	Name    string `json:"stream_name"`
	Type    string `json:"type"`
	URL     string `json:"url"`
	Quality int    `json:"quality"`
}

func (s *stream) low() string {
	return s.URL
}

func (s *stream) high() string {
	return strings.Replace(s.URL, "_low", "", 1)
}
