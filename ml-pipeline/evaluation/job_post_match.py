#!/usr/bin/env python3
"""Post-match evaluation job - compares predictions to actual results."""

from __future__ import annotations

import os
import sys

import psycopg2
from dotenv import load_dotenv

load_dotenv()


def main():
    db_url = os.getenv("DATABASE_URL", "postgres://prono:prono_secret@localhost:5432/prono")
    # psycopg2 uses postgresql not postgres
    db_url = db_url.replace("postgres://", "postgresql://", 1)

    conn = psycopg2.connect(db_url)
    cur = conn.cursor()

    cur.execute("""
        INSERT INTO prediction_outcomes (prediction_id, market, predicted_probability, actual_outcome, is_correct)
        SELECT p.id, key, (p.predictions->>key)::decimal,
            CASE WHEN key = 'home_win' THEN (m.home_score > m.away_score)
                WHEN key = 'draw' THEN (m.home_score = m.away_score)
                WHEN key = 'away_win' THEN (m.home_score < m.away_score)
                WHEN key = 'over_2_5' THEN ((m.home_score + m.away_score) > 2)
                WHEN key = 'btts' THEN (m.home_score > 0 AND m.away_score > 0)
                ELSE false END,
            CASE WHEN (p.predictions->>key)::decimal >= 0.5 THEN
                CASE WHEN key = 'home_win' THEN (m.home_score > m.away_score)
                    WHEN key = 'draw' THEN (m.home_score = m.away_score)
                    WHEN key = 'away_win' THEN (m.home_score < m.away_score)
                    WHEN key = 'over_2_5' THEN ((m.home_score + m.away_score) > 2)
                    WHEN key = 'btts' THEN (m.home_score > 0 AND m.away_score > 0)
                    ELSE false END
            ELSE NOT CASE WHEN key = 'home_win' THEN (m.home_score > m.away_score)
                WHEN key = 'draw' THEN (m.home_score = m.away_score)
                WHEN key = 'away_win' THEN (m.home_score < m.away_score)
                WHEN key = 'over_2_5' THEN ((m.home_score + m.away_score) > 2)
                WHEN key = 'btts' THEN (m.home_score > 0 AND m.away_score > 0)
                ELSE false END END
        FROM predictions p
        JOIN matches m ON m.id = p.match_id
        CROSS JOIN LATERAL jsonb_object_keys(p.predictions) AS key
        WHERE m.status = 'finished' AND m.home_score IS NOT NULL
        AND NOT EXISTS (
            SELECT 1 FROM prediction_outcomes po
            WHERE po.prediction_id = p.id AND po.market = key
        )
        ON CONFLICT DO NOTHING
    """)
    count = cur.rowcount

    # Update rolling performance metrics
    cur.execute("""
        INSERT INTO model_performance (model_name, model_version, market, accuracy, sample_size, period_start, period_end)
        SELECT
            split_part(p.model_version, '-', 1),
            p.model_version,
            po.market,
            AVG(CASE WHEN po.is_correct THEN 1.0 ELSE 0.0 END),
            COUNT(*),
            MIN(m.kickoff_at::date),
            MAX(m.kickoff_at::date)
        FROM prediction_outcomes po
        JOIN predictions p ON p.id = po.prediction_id
        JOIN matches m ON m.id = p.match_id
        GROUP BY p.model_version, po.market
        ON CONFLICT DO NOTHING
    """)

    conn.commit()
    cur.close()
    conn.close()

    print(f"Evaluated {count} prediction outcomes")
    return 0


if __name__ == "__main__":
    sys.exit(main())
