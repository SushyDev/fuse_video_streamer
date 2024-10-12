package real_debrid

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

type File struct {
	ID       int    `json:"id"`
	Path     string `json:"path"`
	Bytes    int64  `json:"bytes"`
	Selected int    `json:"selected"`
}

type TorrentInfoResponse struct {
	ID               string   `json:"id"`
	Filename         string   `json:"filename"`
	OriginalFilename string   `json:"original_filename"`
	Hash             string   `json:"hash"`
	Bytes            int64    `json:"bytes"`
	Host             string   `json:"host"`
	Split            int      `json:"split"`
	Progress         int      `json:"progress"`
	Status           string   `json:"status"`
	Added            string   `json:"added"`
	Files            []File   `json:"files"`
	Links            []string `json:"links"`
	Ended            string   `json:"ended"`
}

func GetTorrentInfo(id string) (*TorrentInfoResponse, error) {
	url, _ := url.Parse(apiHost)

	url.Path += apiPath + "/torrents/info/" + id

	req, err := http.NewRequest("GET", url.String(), nil)
	if err != nil {
		return nil, err
	}

	response, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	defer response.Body.Close()

	switch response.StatusCode {
	case 200:
		var torrentInfo = &TorrentInfoResponse{}
		if err := json.NewDecoder(response.Body).Decode(torrentInfo); err != nil {
			return nil, err
		}

		return torrentInfo, nil
	case 401:
		return nil, fmt.Errorf("Bad token (expired, invalid)")
	case 403:
		return nil, fmt.Errorf("Permission denied (account locked, not premium)")
	case 404:
		return nil, fmt.Errorf("Unknown resource (invalid id): %s", "0")
	default:
		return nil, fmt.Errorf("Unknown error")
	}
}
