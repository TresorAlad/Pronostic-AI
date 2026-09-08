package apifootball

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type Client struct {
	baseURL    string
	host       string
	apiKey     string
	authMode   string
	httpClient *http.Client
	lastCall   time.Time
	minDelay   time.Duration
}

func NewClient(baseURL, host, apiKey, authMode string) *Client {
	if authMode == "" {
		authMode = "apisports"
	}
	return &Client{
		baseURL:    baseURL,
		host:       host,
		apiKey:     apiKey,
		authMode:   authMode,
		httpClient: &http.Client{Timeout: 30 * time.Second},
		minDelay:   250 * time.Millisecond,
	}
}

func (c *Client) Enabled() bool {
	return c != nil && c.apiKey != ""
}

func (c *Client) doRequest(ctx context.Context, path string, params map[string]string) ([]byte, error) {
	elapsed := time.Since(c.lastCall)
	if elapsed < c.minDelay {
		time.Sleep(c.minDelay - elapsed)
	}
	c.lastCall = time.Now()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return nil, err
	}
	if c.authMode == "rapidapi" {
		req.Header.Set("x-rapidapi-key", c.apiKey)
		req.Header.Set("x-rapidapi-host", c.host)
	} else {
		req.Header.Set("x-apisports-key", c.apiKey)
	}

	q := req.URL.Query()
	for k, v := range params {
		q.Set(k, v)
	}
	req.URL.RawQuery = q.Encode()

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}
	return body, nil
}

type Fixture struct {
	Fixture struct {
		ID     int    `json:"id"`
		Date   string `json:"date"`
		Status struct {
			Short string `json:"short"`
		} `json:"status"`
	} `json:"fixture"`
	League struct {
		Name  string `json:"name"`
		Round string `json:"round"`
	} `json:"league"`
	Teams struct {
		Home TeamInfo `json:"home"`
		Away TeamInfo `json:"away"`
	} `json:"teams"`
	Goals struct {
		Home *int `json:"home"`
		Away *int `json:"away"`
	} `json:"goals"`
}

type TeamInfo struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

func finishedStatuses() map[string]bool {
	return map[string]bool{
		"FT": true, "AET": true, "PEN": true, "AWD": true, "WO": true,
	}
}

func (c *Client) GetLastFixturesByTeam(ctx context.Context, teamID, last int) ([]Fixture, error) {
	if last <= 0 {
		last = 10
	}
	body, err := c.doRequest(ctx, "/fixtures", map[string]string{
		"team": fmt.Sprintf("%d", teamID),
		"last": fmt.Sprintf("%d", last),
	})
	if err != nil {
		return nil, err
	}
	var resp struct {
		Response []Fixture       `json:"response"`
		Errors   json.RawMessage `json:"errors"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}
	if len(resp.Errors) > 0 && string(resp.Errors) != "[]" && string(resp.Errors) != "{}" && string(resp.Errors) != "null" {
		return nil, fmt.Errorf("API-Football: %s", string(resp.Errors))
	}
	out := make([]Fixture, 0, len(resp.Response))
	for _, f := range resp.Response {
		if finishedStatuses()[strings.ToUpper(f.Fixture.Status.Short)] {
			out = append(out, f)
		}
	}
	return out, nil
}

func (c *Client) GetHeadToHead(ctx context.Context, homeTeamID, awayTeamID, last int) ([]Fixture, error) {
	if last <= 0 {
		last = 8
	}
	body, err := c.doRequest(ctx, "/fixtures/headtohead", map[string]string{
		"h2h":  fmt.Sprintf("%d-%d", homeTeamID, awayTeamID),
		"last": fmt.Sprintf("%d", last),
	})
	if err != nil {
		return nil, err
	}
	var resp struct {
		Response []Fixture `json:"response"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}
	return resp.Response, nil
}
