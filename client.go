package honeycombio

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
)

// Config holds all configuration options for the client.
type Config struct {
	// Required - the API key to use when sending request to Honeycomb.
	APIKey string
	// URL of the Honeycomb API, defaults to "https://api.honeycomb.io".
	APIUrl string
	// User agent to send with all requests, defaults to "go-honeycombio".
	UserAgent string
}

func defaultConfig() *Config {
	return &Config{
		APIKey:    "",
		APIUrl:    "https://api.honeycomb.io",
		UserAgent: "go-honeycombio",
	}
}

// Merge the given config by copying all non-blank values.
func (c *Config) merge(other *Config) {
	if other.APIKey != "" {
		c.APIKey = other.APIKey
	}
	if other.APIUrl != "" {
		c.APIUrl = other.APIUrl
	}
	if other.UserAgent != "" {
		c.UserAgent = other.UserAgent
	}
}

// Client to interact with Honeycomb.
type Client struct {
	apiKey     string
	apiURL     *url.URL
	userAgent  string
	httpClient *http.Client

	Boards   Boards
	Markers  Markers
	Triggers Triggers
}

// NewClient creates a new Honeycomb API client.
func NewClient(config *Config) (*Client, error) {
	cfg := defaultConfig()
	cfg.merge(config)

	if cfg.APIKey == "" {
		return nil, errors.New("APIKey must be configured")
	}
	apiURL, err := url.Parse(cfg.APIUrl)
	if err != nil {
		return nil, fmt.Errorf("could not parse APIUrl: %w", err)
	}

	client := &Client{
		apiKey:     cfg.APIKey,
		apiURL:     apiURL,
		userAgent:  cfg.UserAgent,
		httpClient: &http.Client{},
	}
	client.Boards = &boards{client: client}
	client.Markers = &markers{client: client}
	client.Triggers = &triggers{client: client}

	return client, nil
}

// ErrNotFound means that the requested item could not be found.
var ErrNotFound = errors.New("request failed with status code 404")

// newRequest prepares a request to the Honeycomb API with the default Honeycomb
// headers and a JSON body, if v is set.
func (c *Client) newRequest(method, path string, v interface{}) (*http.Request, error) {
	var body io.Reader

	if v != nil {
		buf := new(bytes.Buffer)
		err := json.NewEncoder(buf).Encode(v)
		if err != nil {
			return nil, err
		}
		body = buf
	}

	requestURL, err := c.apiURL.Parse(path)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(method, requestURL.String(), body)
	if err != nil {
		return nil, err
	}

	req.Header.Add("X-Honeycomb-Team", c.apiKey)
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("User-Agent", c.userAgent)

	return req, nil
}

// do a request and parse the response in v, if v is not nil. Returns an error
// if the request failed or if the response contained a non-2xx status code.
// ErrNotFound is returned on a 404 response.
func (c *Client) do(req *http.Request, v interface{}) error {
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if !is2xx(resp.StatusCode) {
		if resp.StatusCode == 404 {
			return ErrNotFound
		}

		errorMsg := attemptToExtractHoneycombioError(resp.Body)
		if errorMsg == "" {
			return fmt.Errorf("request failed (%d)", resp.StatusCode)
		}
		return fmt.Errorf("request failed (%d): %s", resp.StatusCode, errorMsg)
	}

	if v != nil {
		err = json.NewDecoder(resp.Body).Decode(v)
	}
	return err
}

func is2xx(status int) bool {
	return status >= 200 && status < 300
}

type honeycombioError struct {
	ErrorMessage string `json:"error"`
}

func attemptToExtractHoneycombioError(bodyReader io.Reader) string {
	body, err := ioutil.ReadAll(bodyReader)
	if err != nil {
		return ""
	}

	var honeycombioErr honeycombioError

	err = json.Unmarshal(body, &honeycombioErr)
	if err != nil || honeycombioErr.ErrorMessage == "" {
		return string(body)
	}

	return honeycombioErr.ErrorMessage
}

// urlEncodeDataset sanitizes the dataset name for when it is used as part of
// the URL.
func urlEncodeDataset(dataset string) string {
	return strings.Replace(dataset, "/", "-", -1)
}
