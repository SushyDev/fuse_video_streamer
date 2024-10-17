package real_debrid

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"time"

	"golang.org/x/time/rate"
)

type RealDebridClient struct {
	Token       string
	Client      http.Client
	RateLimiter *rate.Limiter
}

func NewRealDebridClient() *RealDebridClient {
	flag.Parse()
	token := flag.Arg(1)

	return &RealDebridClient{
		Token:       token,
		Client:      http.Client{},
		RateLimiter: rate.NewLimiter(rate.Every(time.Second*60), 245),
	}
}

func (c *RealDebridClient) Do(req *http.Request) (*http.Response, error) {
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.Token))

	ctx := context.Background()
	err := c.RateLimiter.Wait(ctx)
	if err != nil {
		return nil, err
	}

	return c.Client.Do(req)
}

type AddMagnetResponse struct {
	Id  string `json:"id"`
	Uri string `json:"uri"`
}

var apiHost = "https://api.real-debrid.com"
var apiPath = "/rest/1.0"

// var settings = config.GetSettings()

// var sugar = logger.Sugar()

var client = NewRealDebridClient()
