package api

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
)

var UrlRegex = regexp.MustCompile("^https?://(.*)/")

type errorObject struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type errorResponse struct {
	Errors []errorObject `json:"Errors"`
}

type Client struct {
	ClientUrl string
	baseUrl  string
	username string
	password string

	HTTPClient *http.Client
}

func NewClient(url, username, password string) *Client {
	return &Client{
		ClientUrl: url,
		baseUrl:  url + "/api/v2.0",
		username: username,
		password: password,

		HTTPClient: &http.Client{},
	}
}

func (c *Client) CreateRequest(method, url string, body io.Reader) (*http.Request, error) {
	return http.NewRequest(method, c.baseUrl+url, body)
}

func (c *Client) SendRequest(req *http.Request, v interface{}) error {
	req.Header.Add("Accept", "application/json")
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", "Basic "+c.basicAuth())

	// Execute request
	res, err := c.HTTPClient.Do(req)
	if err != nil {
		return err
	}

	// Close request
	defer res.Body.Close()

	// Parse error
	if res.StatusCode < http.StatusOK || res.StatusCode >= http.StatusBadRequest {
		var errRes errorResponse
		if err = json.NewDecoder(res.Body).Decode(&errRes); err == nil {
			return errors.New(errRes.Errors[0].Code + ": " + errRes.Errors[0].Message)
		}

		return fmt.Errorf("unknown error, status code: %d", res.StatusCode)
	}

	if err = json.NewDecoder(res.Body).Decode(&v); err != nil {
		return err
	}

	return nil
}

func (client Client) basicAuth() string {
	auth := client.username + ":" + client.password
	return base64.StdEncoding.EncodeToString([]byte(auth))
}
