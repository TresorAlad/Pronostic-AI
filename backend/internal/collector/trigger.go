package collector

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/redis/go-redis/v9"
)

type Trigger struct {
	baseURL string
	redis   *redis.Client
	client  *http.Client
}

func NewTrigger(baseURL string, redisClient *redis.Client) *Trigger {
	return &Trigger{
		baseURL: baseURL,
		redis:   redisClient,
		client:  &http.Client{Timeout: 90 * time.Second},
	}
}

// RequestDateSync déclenche une sync collector pour une date (dedup Redis 30 min).
func (t *Trigger) RequestDateSync(ctx context.Context, date string) {
	if t == nil || t.baseURL == "" || date == "" {
		return
	}
	if _, err := time.Parse("2006-01-02", date); err != nil {
		return
	}

	if t.redis != nil {
		key := "collector:sync:requested:" + date
		ok, err := t.redis.SetNX(ctx, key, "1", 30*time.Minute).Result()
		if err != nil || !ok {
			return
		}
	}

	go func() {
		url := fmt.Sprintf("%s/sync/date?date=%s", t.baseURL, date)
		req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, url, bytes.NewReader([]byte("{}")))
		if err != nil {
			log.Printf("Collector sync request error: %v", err)
			return
		}
		req.Header.Set("Content-Type", "application/json")
		resp, err := t.client.Do(req)
		if err != nil {
			log.Printf("Collector sync %s failed: %v", date, err)
			return
		}
		resp.Body.Close()
		if resp.StatusCode >= 400 {
			log.Printf("Collector sync %s status %d", date, resp.StatusCode)
			return
		}
		log.Printf("Collector sync triggered for %s", date)
	}()
}
