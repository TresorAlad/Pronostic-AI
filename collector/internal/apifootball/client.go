package apifootball

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type Client struct {
	baseURL    string
	host       string
	apiKey     string
	authMode   string // "apisports" (direct) or "rapidapi"
	httpClient *http.Client
	lastCall   time.Time
	minDelay   time.Duration
}

func NewClient(baseURL, host, apiKey, authMode string) *Client {
	if authMode == "" {
		authMode = "apisports"
	}
	return &Client{
		baseURL:  baseURL,
		host:     host,
		apiKey:   apiKey,
		authMode: authMode,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		minDelay: 200 * time.Millisecond,
	}
}

func (c *Client) doRequest(ctx context.Context, path string, params map[string]string) ([]byte, error) {
	const maxRetries = 4
	var lastErr error

	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			backoff := time.Duration(attempt*attempt) * time.Second
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoff):
			}
		}

		body, status, err := c.doRequestOnce(ctx, path, params)
		if err == nil {
			return body, nil
		}
		lastErr = err
		if status != http.StatusTooManyRequests {
			return nil, err
		}
	}

	return nil, fmt.Errorf("rate limited after retries: %w", lastErr)
}

func (c *Client) doRequestOnce(ctx context.Context, path string, params map[string]string) ([]byte, int, error) {
	elapsed := time.Since(c.lastCall)
	if elapsed < c.minDelay {
		time.Sleep(c.minDelay - elapsed)
	}
	c.lastCall = time.Now()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return nil, 0, err
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
		return nil, 0, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, err
	}

	if resp.StatusCode == http.StatusTooManyRequests {
		return nil, resp.StatusCode, fmt.Errorf("API error 429: %s", string(body))
	}

	if resp.StatusCode != http.StatusOK {
		return nil, resp.StatusCode, fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}

	return body, resp.StatusCode, nil
}

type APIResponse struct {
	Response json.RawMessage `json:"response"`
	Errors   json.RawMessage `json:"errors"`
}

func (c *Client) GetFixtures(ctx context.Context, leagueID, season int) ([]FixtureResponse, error) {
	body, err := c.doRequest(ctx, "/fixtures", map[string]string{
		"league": fmt.Sprintf("%d", leagueID),
		"season": fmt.Sprintf("%d", season),
	})
	if err != nil {
		return nil, err
	}

	var resp struct {
		Response []FixtureResponse `json:"response"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}
	return resp.Response, nil
}

func (c *Client) GetFixturesByDate(ctx context.Context, date string) ([]FixtureResponse, error) {
	body, err := c.doRequest(ctx, "/fixtures", map[string]string{
		"date": date,
	})
	if err != nil {
		return nil, err
	}

	var resp struct {
		Response []FixtureResponse `json:"response"`
		Errors   json.RawMessage   `json:"errors"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}
	if len(resp.Errors) > 0 && string(resp.Errors) != "[]" && string(resp.Errors) != "{}" && string(resp.Errors) != "null" {
		return nil, fmt.Errorf("API-Football: %s", string(resp.Errors))
	}
	return resp.Response, nil
}

func (c *Client) GetLiveFixtures(ctx context.Context) ([]FixtureResponse, error) {
	body, err := c.doRequest(ctx, "/fixtures", map[string]string{
		"live": "all",
	})
	if err != nil {
		return nil, err
	}

	var resp struct {
		Response []FixtureResponse `json:"response"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}
	return resp.Response, nil
}

func (c *Client) GetFixtureStatistics(ctx context.Context, fixtureID int) ([]FixtureStatsResponse, error) {
	body, err := c.doRequest(ctx, "/fixtures/statistics", map[string]string{
		"fixture": fmt.Sprintf("%d", fixtureID),
	})
	if err != nil {
		return nil, err
	}

	var resp struct {
		Response []FixtureStatsResponse `json:"response"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}
	return resp.Response, nil
}

func (c *Client) GetFixtureEvents(ctx context.Context, fixtureID int) ([]EventResponse, error) {
	body, err := c.doRequest(ctx, "/fixtures/events", map[string]string{
		"fixture": fmt.Sprintf("%d", fixtureID),
	})
	if err != nil {
		return nil, err
	}

	var resp struct {
		Response []EventResponse `json:"response"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}
	return resp.Response, nil
}

func (c *Client) GetFixtureLineups(ctx context.Context, fixtureID int) ([]LineupResponse, error) {
	body, err := c.doRequest(ctx, "/fixtures/lineups", map[string]string{
		"fixture": fmt.Sprintf("%d", fixtureID),
	})
	if err != nil {
		return nil, err
	}

	var resp struct {
		Response []LineupResponse `json:"response"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}
	return resp.Response, nil
}

func (c *Client) GetInjuries(ctx context.Context, leagueID, season int) ([]InjuryResponse, error) {
	body, err := c.doRequest(ctx, "/injuries", map[string]string{
		"league": fmt.Sprintf("%d", leagueID),
		"season": fmt.Sprintf("%d", season),
	})
	if err != nil {
		return nil, err
	}

	var resp struct {
		Response []InjuryResponse `json:"response"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}
	return resp.Response, nil
}

func (c *Client) GetOdds(ctx context.Context, fixtureID int) ([]OddsResponse, error) {
	body, err := c.doRequest(ctx, "/odds", map[string]string{
		"fixture": fmt.Sprintf("%d", fixtureID),
	})
	if err != nil {
		return nil, err
	}
	var resp struct {
		Response []OddsResponse `json:"response"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}
	return resp.Response, nil
}

// Response types

type FixtureResponse struct {
	Fixture struct {
		ID        int    `json:"id"`
		Referee   string `json:"referee"`
		Timezone  string `json:"timezone"`
		Date      string `json:"date"`
		Timestamp int64  `json:"timestamp"`
		Venue     struct {
			Name string `json:"name"`
		} `json:"venue"`
		Status struct {
			Long    string `json:"long"`
			Short   string `json:"short"`
			Elapsed int    `json:"elapsed"`
		} `json:"status"`
	} `json:"fixture"`
	League struct {
		ID     int    `json:"id"`
		Name   string `json:"name"`
		Country string `json:"country"`
		Logo   string `json:"logo"`
		Season int    `json:"season"`
		Round  string `json:"round"`
	} `json:"league"`
	Teams struct {
		Home TeamInfo `json:"home"`
		Away TeamInfo `json:"away"`
	} `json:"teams"`
	Goals struct {
		Home *int `json:"home"`
		Away *int `json:"away"`
	} `json:"goals"`
	Score struct {
		Halftime struct {
			Home *int `json:"home"`
			Away *int `json:"away"`
		} `json:"halftime"`
	} `json:"score"`
}

type TeamInfo struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	Logo string `json:"logo"`
}

type FixtureStatsResponse struct {
	Team   TeamInfo      `json:"team"`
	Statistics []StatItem `json:"statistics"`
}

type StatItem struct {
	Type  string      `json:"type"`
	Value interface{} `json:"value"`
}

type EventResponse struct {
	Time struct {
		Elapsed  int  `json:"elapsed"`
		Extra    *int `json:"extra"`
	} `json:"time"`
	Team   TeamInfo `json:"team"`
	Player struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	} `json:"player"`
	Assist struct {
		ID   *int   `json:"id"`
		Name string `json:"name"`
	} `json:"assist"`
	Type   string `json:"type"`
	Detail string `json:"detail"`
}

type LineupResponse struct {
	Team    TeamInfo `json:"team"`
	Coach   struct {
		Name string `json:"name"`
	} `json:"coach"`
	Formation string `json:"formation"`
	StartXI   []struct {
		Player struct {
			ID     int    `json:"id"`
			Name   string `json:"name"`
			Number int    `json:"number"`
			Pos    string `json:"pos"`
			Grid   string `json:"grid"`
		} `json:"player"`
	} `json:"startXI"`
	Substitutes []struct {
		Player struct {
			ID     int    `json:"id"`
			Name   string `json:"name"`
			Number int    `json:"number"`
			Pos    string `json:"pos"`
		} `json:"player"`
	} `json:"substitutes"`
}

type InjuryResponse struct {
	Player struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	} `json:"player"`
	Team TeamInfo `json:"team"`
	Fixture struct {
		ID int `json:"id"`
	} `json:"fixture"`
	League struct {
		ID     int `json:"id"`
		Season int `json:"season"`
	} `json:"league"`
	Type  string `json:"type"`
	Reason string `json:"reason"`
}

type OddsResponse struct {
	Bookmakers []struct {
		Name string `json:"name"`
		Bets []struct {
			Name   string `json:"name"`
			Values []struct {
				Value string `json:"value"`
				Odd   string `json:"odd"`
			} `json:"values"`
		} `json:"bets"`
	} `json:"bookmakers"`
}
