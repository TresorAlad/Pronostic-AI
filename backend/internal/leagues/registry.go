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
		sort.Slice(cfg.Leagues, func(i, j int) bool {
			return cfg.Leagues[i].Priority < cfg.Leagues[j].Priority
		})
		cached = &cfg
	})
	return cached, loadErr
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
	ids := make([]int, len(c.Leagues))
	for i, l := range c.Leagues {
		ids[i] = l.ExternalID
	}
	return ids
}

func (c *Config) PrimaryExternalIDs() []int {
	return c.TrackedExternalIDs()
}

func (c *Config) CouponExternalIDs() []int {
	seen := make(map[int]struct{})
	var ids []int
	for _, id := range c.TrackedExternalIDs() {
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
	}
	for _, id := range c.CouponExtraExternalIDs {
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
	}
	return ids
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
	ids := c.PrimaryExternalIDs()
	if len(ids) == 0 {
		return "1"
	}
	idSet := make(map[int]struct{}, len(ids))
	for _, id := range ids {
		idSet[id] = struct{}{}
	}
	sql := fmt.Sprintf("CASE WHEN %s = ANY(ARRAY[%s]) THEN 0 ELSE 1 END", column, intSliceCSV(ids))
	return sql
}

func intSliceCSV(ids []int) string {
	if len(ids) == 0 {
		return ""
	}
	out := fmt.Sprintf("%d", ids[0])
	for _, id := range ids[1:] {
		out += fmt.Sprintf(",%d", id)
	}
	return out
}

func (c *Config) Labels() []string {
	labels := make([]string, len(c.Leagues))
	for i, l := range c.Leagues {
		labels[i] = l.Label
	}
	return labels
}
