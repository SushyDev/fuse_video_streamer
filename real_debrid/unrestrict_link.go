package real_debrid

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	urlEncoding "net/url"
	"strings"
)

type UnrestrictLinkResponse struct {
	ID         string `json:"id"`
	Filename   string `json:"filename"`
	MimeType   string `json:"mimeType"`
	FileSize   int64  `json:"fileSize"`
	Link       string `json:"link"`
	Host       string `json:"host"`
	Chunks     int    `json:"chunks"`
	Crc        int    `json:"crc"`
	Download   string `json:"download"`
	Streamable int    `json:"streamable"`
}

func UnrestrictLink(link string) (*UnrestrictLinkResponse, error) {
	url, _ := url.Parse(apiHost)

	url.Path += apiPath + "/unrestrict/link"

	form := urlEncoding.Values{}
	form.Add("link", link)

	formReader := strings.NewReader(form.Encode())

	req, err := http.NewRequest("POST", url.String(), formReader)
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
		var data = &UnrestrictLinkResponse{}
		if err := json.NewDecoder(response.Body).Decode(data); err != nil {
			return nil, err
		}

		return data, nil
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
