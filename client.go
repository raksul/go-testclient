package testclient

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
)

type Client struct {
	server   http.Handler
	response *http.Response
}

func New(server http.Handler) *Client {
	return &Client{
		server: server,
	}
}

func (c *Client) Request(req *http.Request) {
	rec := httptest.NewRecorder()
	c.server.ServeHTTP(rec, req)
	c.response = rec.Result()
}

func (c *Client) PostForm(uri string, params map[string]string) {
	p := url.Values{}
	for key, value := range params {
		p.Add(key, value)
	}
	form := strings.NewReader(p.Encode())

	req := httptest.NewRequest(http.MethodPost, uri, form)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	c.Request(req)
}

func (c *Client) FollowRedirect() error {
	// check redirect conditions
	if !(300 <= c.response.StatusCode && c.response.StatusCode < 400) {
		return fmt.Errorf("bad http status code for redirect: %d", c.response.StatusCode)
	}
	location := c.response.Header.Get("Location")
	if location == "" {
		return fmt.Errorf("no Location header error")
	}

	cookie := c.response.Header.Get("Set-Cookie")
	req := httptest.NewRequest(http.MethodGet, location, nil)
	req.Header.Set("Cookie", cookie)
	c.Request(req)

	return nil
}

func (c *Client) Response() *http.Response {
	return c.response
}
