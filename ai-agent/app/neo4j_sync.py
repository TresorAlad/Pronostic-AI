"""Neo4j knowledge graph sync and queries."""

from __future__ import annotations

import os

from neo4j import GraphDatabase


class KnowledgeGraph:
    def __init__(self):
        uri = os.getenv("NEO4J_URI", "bolt://localhost:7687")
        user = os.getenv("NEO4J_USER", "neo4j")
        password = os.getenv("NEO4J_PASSWORD", "neo4j_secret")
        self.driver = GraphDatabase.driver(uri, auth=(user, password))

    def close(self):
        self.driver.close()

    def init_schema(self):
        with self.driver.session() as session:
            session.run("CREATE CONSTRAINT IF NOT EXISTS FOR (c:Club) REQUIRE c.id IS UNIQUE")
            session.run("CREATE CONSTRAINT IF NOT EXISTS FOR (p:Player) REQUIRE p.id IS UNIQUE")
            session.run("CREATE CONSTRAINT IF NOT EXISTS FOR (m:Match) REQUIRE m.id IS UNIQUE")

    def sync_club(self, club_id: str, name: str):
        with self.driver.session() as session:
            session.run(
                "MERGE (c:Club {id: $id}) SET c.name = $name",
                id=club_id, name=name,
            )

    def sync_player(self, player_id: str, name: str, club_id: str):
        with self.driver.session() as session:
            session.run(
                """
                MERGE (p:Player {id: $pid}) SET p.name = $name
                WITH p
                MATCH (c:Club {id: $cid})
                MERGE (c)-[:HAS_PLAYER]->(p)
                """,
                pid=player_id, name=name, cid=club_id,
            )

    def sync_match(self, match_id: str, home_id: str, away_id: str, date: str):
        with self.driver.session() as session:
            session.run(
                """
                MERGE (m:Match {id: $mid}) SET m.date = $date
                WITH m
                MATCH (h:Club {id: $hid}), (a:Club {id: $aid})
                MERGE (m)-[:HOME_TEAM]->(h)
                MERGE (m)-[:AWAY_TEAM]->(a)
                """,
                mid=match_id, hid=home_id, aid=away_id, date=date,
            )

    def get_h2h_context(self, home_id: str, away_id: str, limit: int = 5) -> list:
        with self.driver.session() as session:
            result = session.run(
                """
                MATCH (m:Match)-[:HOME_TEAM]->(h:Club {id: $hid})
                MATCH (m)-[:AWAY_TEAM]->(a:Club {id: $aid})
                RETURN m.id AS id, m.date AS date
                ORDER BY m.date DESC LIMIT $limit
                """,
                hid=home_id, aid=away_id, limit=limit,
            )
            return [dict(r) for r in result]

    def mark_injured(self, player_id: str, reason: str):
        with self.driver.session() as session:
            session.run(
                """
                MATCH (p:Player {id: $pid})
                MERGE (s:Status {type: 'injured', reason: $reason})
                MERGE (p)-[:INJURED]->(s)
                """,
                pid=player_id, reason=reason,
            )
