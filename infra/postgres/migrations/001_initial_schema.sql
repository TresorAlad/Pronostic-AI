-- Football AI Predictor - Initial Schema
-- Anti-leakage: features use data_snapshot_at; match stats indexed by kickoff_at

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Leagues & Seasons
CREATE TABLE leagues (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    external_id INTEGER UNIQUE NOT NULL,
    name VARCHAR(255) NOT NULL,
    country VARCHAR(100),
    logo_url TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE seasons (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    league_id UUID NOT NULL REFERENCES leagues(id) ON DELETE CASCADE,
    year INTEGER NOT NULL,
    start_date DATE,
    end_date DATE,
    is_current BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(league_id, year)
);

-- Teams & Players
CREATE TABLE teams (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    external_id INTEGER UNIQUE NOT NULL,
    name VARCHAR(255) NOT NULL,
    code VARCHAR(10),
    country VARCHAR(100),
    logo_url TEXT,
    founded INTEGER,
    venue_name VARCHAR(255),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE players (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    external_id INTEGER UNIQUE NOT NULL,
    name VARCHAR(255) NOT NULL,
    firstname VARCHAR(100),
    lastname VARCHAR(100),
    age INTEGER,
    nationality VARCHAR(100),
    height VARCHAR(20),
    weight VARCHAR(20),
    photo_url TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE team_players (
    team_id UUID NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    player_id UUID NOT NULL REFERENCES players(id) ON DELETE CASCADE,
    season_id UUID REFERENCES seasons(id) ON DELETE SET NULL,
    position VARCHAR(50),
    number INTEGER,
    PRIMARY KEY (team_id, player_id, season_id)
);

-- Matches
CREATE TYPE match_status AS ENUM (
    'scheduled', 'live', 'finished', 'postponed', 'cancelled', 'suspended'
);

CREATE TABLE matches (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    external_id INTEGER UNIQUE NOT NULL,
    league_id UUID NOT NULL REFERENCES leagues(id),
    season_id UUID REFERENCES seasons(id),
    home_team_id UUID NOT NULL REFERENCES teams(id),
    away_team_id UUID NOT NULL REFERENCES teams(id),
    kickoff_at TIMESTAMPTZ NOT NULL,
    status match_status NOT NULL DEFAULT 'scheduled',
    minute INTEGER,
    home_score INTEGER,
    away_score INTEGER,
    home_score_ht INTEGER,
    away_score_ht INTEGER,
    venue VARCHAR(255),
    referee VARCHAR(255),
    round VARCHAR(100),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_matches_kickoff_at ON matches(kickoff_at);
CREATE INDEX idx_matches_status ON matches(status);
CREATE INDEX idx_matches_league_kickoff ON matches(league_id, kickoff_at);

-- Match Statistics
CREATE TABLE match_statistics (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    match_id UUID NOT NULL REFERENCES matches(id) ON DELETE CASCADE,
    team_id UUID NOT NULL REFERENCES teams(id),
    shots_on_goal INTEGER,
    shots_off_goal INTEGER,
    total_shots INTEGER,
    blocked_shots INTEGER,
    shots_inside_box INTEGER,
    shots_outside_box INTEGER,
    fouls INTEGER,
    corner_kicks INTEGER,
    offsides INTEGER,
    ball_possession DECIMAL(5,2),
    yellow_cards INTEGER,
    red_cards INTEGER,
    goalkeeper_saves INTEGER,
    total_passes INTEGER,
    passes_accurate INTEGER,
    passes_pct DECIMAL(5,2),
    expected_goals DECIMAL(5,2),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(match_id, team_id)
);

-- Match Events
CREATE TYPE event_type AS ENUM (
    'goal', 'own_goal', 'penalty', 'missed_penalty',
    'yellow_card', 'red_card', 'substitution', 'var'
);

CREATE TABLE match_events (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    match_id UUID NOT NULL REFERENCES matches(id) ON DELETE CASCADE,
    team_id UUID REFERENCES teams(id),
    player_id UUID REFERENCES players(id),
    assist_player_id UUID REFERENCES players(id),
    event_type event_type NOT NULL,
    detail VARCHAR(100),
    minute INTEGER,
    extra_minute INTEGER,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_match_events_match ON match_events(match_id);

-- Lineups
CREATE TABLE lineups (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    match_id UUID NOT NULL REFERENCES matches(id) ON DELETE CASCADE,
    team_id UUID NOT NULL REFERENCES teams(id),
    formation VARCHAR(20),
    coach_name VARCHAR(255),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(match_id, team_id)
);

CREATE TABLE lineup_players (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    lineup_id UUID NOT NULL REFERENCES lineups(id) ON DELETE CASCADE,
    player_id UUID NOT NULL REFERENCES players(id),
    is_starter BOOLEAN NOT NULL DEFAULT TRUE,
    position VARCHAR(50),
    grid VARCHAR(10),
    UNIQUE(lineup_id, player_id)
);

-- Player Match Stats
CREATE TABLE player_match_stats (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    match_id UUID NOT NULL REFERENCES matches(id) ON DELETE CASCADE,
    player_id UUID NOT NULL REFERENCES players(id),
    team_id UUID NOT NULL REFERENCES teams(id),
    minutes_played INTEGER,
    rating DECIMAL(4,2),
    goals INTEGER DEFAULT 0,
    assists INTEGER DEFAULT 0,
    shots_total INTEGER DEFAULT 0,
    shots_on_target INTEGER DEFAULT 0,
    passes_total INTEGER DEFAULT 0,
    passes_key INTEGER DEFAULT 0,
    tackles INTEGER DEFAULT 0,
    duels_won INTEGER DEFAULT 0,
    dribbles_attempts INTEGER DEFAULT 0,
    dribbles_success INTEGER DEFAULT 0,
    fouls_committed INTEGER DEFAULT 0,
    fouls_drawn INTEGER DEFAULT 0,
    yellow_cards INTEGER DEFAULT 0,
    red_cards INTEGER DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(match_id, player_id)
);

-- Injuries & Suspensions
CREATE TABLE injuries (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    player_id UUID NOT NULL REFERENCES players(id),
    team_id UUID NOT NULL REFERENCES teams(id),
    reason VARCHAR(255),
    type VARCHAR(100),
    start_date DATE,
    expected_return DATE,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE suspensions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    player_id UUID NOT NULL REFERENCES players(id),
    team_id UUID NOT NULL REFERENCES teams(id),
    reason VARCHAR(255),
    matches_remaining INTEGER,
    start_date DATE,
    end_date DATE,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Users & Auth
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    display_name VARCHAR(100),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Predictions
CREATE TABLE predictions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    match_id UUID NOT NULL REFERENCES matches(id) ON DELETE CASCADE,
    model_version VARCHAR(50) NOT NULL,
    predictions JSONB NOT NULL,
    confidence JSONB NOT NULL,
    no_bet_recommended BOOLEAN DEFAULT FALSE,
    data_snapshot_at TIMESTAMPTZ NOT NULL,
    is_live BOOLEAN DEFAULT FALSE,
    ai_analysis TEXT,
    ai_reasons JSONB,
    ai_abstain BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_predictions_match ON predictions(match_id);
CREATE INDEX idx_predictions_snapshot ON predictions(data_snapshot_at);

-- Prediction Outcomes (evaluation)
CREATE TABLE prediction_outcomes (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    prediction_id UUID NOT NULL REFERENCES predictions(id) ON DELETE CASCADE,
    market VARCHAR(50) NOT NULL,
    predicted_probability DECIMAL(5,4) NOT NULL,
    actual_outcome BOOLEAN NOT NULL,
    is_correct BOOLEAN NOT NULL,
    evaluated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(prediction_id, market)
);

-- Saved Coupons
CREATE TABLE saved_coupons (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    name VARCHAR(255),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE coupon_selections (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    coupon_id UUID NOT NULL REFERENCES saved_coupons(id) ON DELETE CASCADE,
    match_id UUID NOT NULL REFERENCES matches(id),
    market VARCHAR(50) NOT NULL,
    selection VARCHAR(100) NOT NULL,
    confidence DECIMAL(5,4) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Model Performance Metrics
CREATE TABLE model_performance (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    model_name VARCHAR(50) NOT NULL,
    model_version VARCHAR(50) NOT NULL,
    market VARCHAR(50) NOT NULL,
    accuracy DECIMAL(5,4),
    precision_score DECIMAL(5,4),
    recall_score DECIMAL(5,4),
    f1_score DECIMAL(5,4),
    log_loss DECIMAL(8,6),
    brier_score DECIMAL(8,6),
    sample_size INTEGER,
    period_start DATE,
    period_end DATE,
    calculated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_model_performance_name ON model_performance(model_name, market);

-- Seed Top 5 European Leagues (API-Football external IDs)
INSERT INTO leagues (external_id, name, country) VALUES
    (39, 'Premier League', 'England'),
    (140, 'La Liga', 'Spain'),
    (135, 'Serie A', 'Italy'),
    (78, 'Bundesliga', 'Germany'),
    (61, 'Ligue 1', 'France');
