package leagues

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
)

type Entry struct {
	ExternalID int    `json:"external_id"`
	Label      string `json:"label"`
	Priority   int    `json:"priority"`
	Tier       string `json:"tier,omitempty"` // continental, top5, secondary
}

type Config struct {
	topMatchesMax          int
	couponMatchPoolMax     int
	CouponExtraExternalIDs []int   `json:"coupon_extra_external_ids"`
	Leagues                []Entry `json:"leagues"`
}

type configJSON struct {
	TopMatchesMax          int     `json:"top_matches_max"`
	CouponMatchPoolMax     int     `json:"coupon_match_pool_max"`
	CouponExtraExternalIDs []int   `json:"coupon_extra_external_ids"`
	Leagues                []Entry `json:"leagues"`
}

var (
	loadOnce sync.Once
	cached   *Config
	loadErr  error
)

func configPath() string {
	if p := os.Getenv("LEAGUES_CONFIG_PATH"); p != "" {
		return p
	}
	candidates := []string{
		"config/leagues.json",
		"../config/leagues.json",
		"../../config/leagues.json",
	}
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			abs, err := filepath.Abs(c)
			if err == nil {
				return abs
			}
			return c
		}
	}
	return "config/leagues.json"
}

func Load() (*Config, error) {
	loadOnce.Do(func() {
		path := configPath()
		data, err := os.ReadFile(path)
		if err != nil {
			loadErr = fmt.Errorf("read leagues config %s: %w", path, err)
			return
		}
		var raw configJSON
		if err := json.Unmarshal(data, &raw); err != nil {
			loadErr = fmt.Errorf("parse leagues config: %w", err)
			return
		}
		cfg := Config{
			topMatchesMax:          raw.TopMatchesMax,
			couponMatchPoolMax:     raw.CouponMatchPoolMax,
			CouponExtraExternalIDs: raw.CouponExtraExternalIDs,
			Leagues:                raw.Leagues,
		}
		if cfg.topMatchesMax <= 0 {
			cfg.topMatchesMax = 10
		}
		if cfg.couponMatchPoolMax <= 0 {
			cfg.couponMatchPoolMax = 200
		}
		if len(cfg.Leagues) == 0 {
			loadErr = fmt.Errorf("leagues config: no leagues defined")
			return
		}
		cfg.mergeLegacyExtras()
		sort.Slice(cfg.Leagues, func(i, j int) bool {
			return cfg.Leagues[i].Priority < cfg.Leagues[j].Priority
		})
		cached = &cfg
	})
	return cached, loadErr
}

func (c *Config) mergeLegacyExtras() {
	if len(c.CouponExtraExternalIDs) == 0 {
		return
	}
	seen := make(map[int]struct{}, len(c.Leagues))
	maxPriority := 0
	for _, l := range c.Leagues {
		seen[l.ExternalID] = struct{}{}
		if l.Priority > maxPriority {
			maxPriority = l.Priority
		}
	}
	next := maxPriority + 10
	if next < 100 {
		next = 100
	}
	for _, id := range c.CouponExtraExternalIDs {
		if _, ok := seen[id]; ok {
			continue
		}
		c.Leagues = append(c.Leagues, Entry{
			ExternalID: id,
			Label:      fmt.Sprintf("League %d", id),
			Priority:   next,
			Tier:       "secondary",
		})
		seen[id] = struct{}{}
		next += 10
	}
}

func MustLoad() *Config {
	cfg, err := Load()
	if err != nil {
		panic(err)
	}
	return cfg
}

func ResetForTest() {
	loadOnce = sync.Once{}
	cached = nil
	loadErr = nil
}

func (c *Config) TrackedExternalIDs() []int {
	return c.DisplayExternalIDs()
}

func (c *Config) DisplayExternalIDs() []int {
	ids := make([]int, len(c.Leagues))
	for i, l := range c.Leagues {
		ids[i] = l.ExternalID
	}
	return ids
}

func (c *Config) PrimaryExternalIDs() []int {
	var ids []int
	for _, l := range c.Leagues {
		if l.Tier == "top5" {
			ids = append(ids, l.ExternalID)
		}
	}
	if len(ids) == 0 {
		for _, l := range c.Leagues {
			if l.Priority >= 10 && l.Priority <= 14 {
				ids = append(ids, l.ExternalID)
			}
		}
	}
	return ids
}

func (c *Config) PredictableExternalIDs() []int {
	var ids []int
	for _, l := range c.Leagues {
		if l.Tier == "continental" || l.Tier == "top5" {
			ids = append(ids, l.ExternalID)
		}
	}
	if len(ids) == 0 {
		return c.PrimaryExternalIDs()
	}
	return ids
}

func (c *Config) CouponExternalIDs() []int {
	return c.DisplayExternalIDs()
}

func (c *Config) TopMatchesMax() int {
	if c.topMatchesMax <= 0 {
		return 10
	}
	return c.topMatchesMax
}

func (c *Config) CouponMatchPoolMax() int {
	if c.couponMatchPoolMax <= 0 {
		return 200
	}
	return c.couponMatchPoolMax
}

func (c *Config) IsTracked(externalID int) bool {
	for _, l := range c.Leagues {
		if l.ExternalID == externalID {
			return true
		}
	}
	return false
}

func (c *Config) IsCouponLeague(externalID int) bool {
	return c.IsTracked(externalID)
}

func (c *Config) PrioritySQL(column string) string {
	if len(c.Leagues) == 0 {
		return "999"
	}
	sql := "CASE"
	for _, l := range c.Leagues {
		sql += fmt.Sprintf(" WHEN %s = %d THEN %d", column, l.ExternalID, l.Priority)
	}
	sql += " ELSE 999 END"
	return sql
}

func (c *Config) PrimaryPrioritySQL(column string) string {
	return c.PrioritySQL(column)
}
func (c *Config) Labels() []string {
	labels := make([]string, len(c.Leagues))
	for i, l := range c.Leagues {
		labels[i] = l.Label
	}
	return labels
}
