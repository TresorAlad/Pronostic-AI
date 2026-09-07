CREATE TABLE IF NOT EXISTS coupon_selection_outcomes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    saved_coupon_id UUID NOT NULL REFERENCES saved_coupons(id) ON DELETE CASCADE,
    match_id UUID NOT NULL REFERENCES matches(id),
    market TEXT NOT NULL,
    selection TEXT NOT NULL,
    predicted_probability DECIMAL,
    actual_outcome BOOLEAN NOT NULL,
    is_correct BOOLEAN NOT NULL,
    evaluated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (saved_coupon_id, match_id, market)
);

CREATE INDEX IF NOT EXISTS idx_coupon_outcomes_coupon ON coupon_selection_outcomes(saved_coupon_id);
