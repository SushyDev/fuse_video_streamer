package real_debrid

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

type Torrent struct {
	ID       string   `json:"id"`
	Filename string   `json:"filename"`
	Hash     string   `json:"hash"`
	Bytes    int64    `json:"bytes"`
	Host     string   `json:"host"`
	Split    int      `json:"split"`
	Progress int      `json:"progress"`
	Status   string   `json:"status"`
	Added    string   `json:"added"`
	Links    []string `json:"links"`
	Ended    string   `json:"ended"`
}

type TorrentsResponse []Torrent

func GetTorrents() (*TorrentsResponse, error) {
	url, _ := url.Parse(apiHost)

	query := url.Query()
	query.Set("limit", "100")
	query.Set("page", "1")

	url.RawQuery = query.Encode()
	url.Path += apiPath + "/torrents"

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
		var torrents = &TorrentsResponse{}
		if err := json.NewDecoder(response.Body).Decode(torrents); err != nil {
			return nil, err
		}

		return torrents, nil
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
