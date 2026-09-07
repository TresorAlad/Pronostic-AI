#!/usr/bin/env python3
"""Sync clubs and matches from PostgreSQL to Neo4j via AI agent API."""

from __future__ import annotations

import os
import sys

import psycopg2
import requests
from dotenv import load_dotenv

load_dotenv()
load_dotenv(os.path.join(os.path.dirname(__file__), "..", ".env"))

DATABASE_URL = os.getenv("DATABASE_URL")
AI_URL = os.getenv("AI_AGENT_URL", "http://localhost:5001")


def main() -> int:
    if not DATABASE_URL:
        print("DATABASE_URL manquant", file=sys.stderr)
        return 1

    conn = psycopg2.connect(DATABASE_URL)
    cur = conn.cursor()

    cur.execute("SELECT id::text, name FROM teams ORDER BY name LIMIT 500")
    clubs = [{"id": row[0], "name": row[1]} for row in cur.fetchall()]

    cur.execute(
        """
        SELECT m.id::text, ht.id::text, at.id::text, m.kickoff_at::text
        FROM matches m
        JOIN teams ht ON ht.id = m.home_team_id
        JOIN teams at ON at.id = m.away_team_id
        ORDER BY m.kickoff_at DESC
        LIMIT 1000
        """
    )
    matches = [
        {"id": row[0], "home_id": row[1], "away_id": row[2], "date": row[3]}
        for row in cur.fetchall()
    ]

    cur.close()
    conn.close()

    resp = requests.post(
        f"{AI_URL.rstrip('/')}/sync/neo4j",
        json={"clubs": clubs, "matches": matches},
        timeout=120,
    )
    if resp.status_code >= 400:
        print(f"Erreur sync Neo4j: {resp.status_code} {resp.text}", file=sys.stderr)
        return 1

    print(f"Sync Neo4j OK: {len(clubs)} clubs, {len(matches)} matchs")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
