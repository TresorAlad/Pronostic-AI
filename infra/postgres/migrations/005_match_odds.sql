CREATE TABLE IF NOT EXISTS match_odds (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    match_id UUID NOT NULL REFERENCES matches(id) ON DELETE CASCADE,
    bookmaker VARCHAR(100) NOT NULL,
    market VARCHAR(100) NOT NULL,
    selection VARCHAR(100) NOT NULL,
    odd DECIMAL(8, 3) NOT NULL,
    fetched_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (match_id, bookmaker, market, selection)
);

CREATE INDEX IF NOT EXISTS idx_match_odds_match ON match_odds(match_id, fetched_at DESC);
