CREATE TABLE IF NOT EXISTS match_odds_history (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    match_id UUID NOT NULL REFERENCES matches(id) ON DELETE CASCADE,
    bookmaker VARCHAR(100) NOT NULL,
    market VARCHAR(100) NOT NULL,
    selection VARCHAR(100) NOT NULL,
    odd DECIMAL(8,4) NOT NULL,
    fetched_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_match_odds_history_lookup
    ON match_odds_history(match_id, market, selection, fetched_at DESC);

ALTER TABLE coupon_selections
    ADD COLUMN IF NOT EXISTS bookmaker_odd DECIMAL(8,4),
    ADD COLUMN IF NOT EXISTS avg_odd DECIMAL(8,4),
    ADD COLUMN IF NOT EXISTS bookmaker_name VARCHAR(100),
    ADD COLUMN IF NOT EXISTS value_edge DECIMAL(8,4),
    ADD COLUMN IF NOT EXISTS odd_trend VARCHAR(10);

ALTER TABLE saved_coupons
    ADD COLUMN IF NOT EXISTS combined_odd DECIMAL(10,4);
