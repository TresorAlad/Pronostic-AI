"""AI Agent Flask service."""

import os

from dotenv import load_dotenv
from flask import Flask, jsonify, request
from flask_cors import CORS

load_dotenv()

from app.agent import analyze
from app.neo4j_sync import KnowledgeGraph

app = Flask(__name__)
CORS(app)

kg = None
try:
    kg = KnowledgeGraph()
    kg.init_schema()
except Exception:
    kg = None


@app.route("/health", methods=["GET"])
def health():
    return jsonify({"status": "ok", "neo4j": kg is not None})


@app.route("/analyze/match", methods=["POST"])
def analyze_match():
    data = request.get_json()
    if not data:
        return jsonify({"error": "JSON body required"}), 400

    if not data.get("predictions"):
        return jsonify({
            "analysis": "Donnees ML manquantes.",
            "reasons": [],
            "recommended_markets": [],
            "abstain": True,
        })

    result = analyze(data)
    return jsonify(result)


@app.route("/sync/neo4j", methods=["POST"])
def sync_neo4j():
    if not kg:
        return jsonify({"error": "Neo4j not connected"}), 503

    data = request.get_json() or {}
    if "clubs" in data:
        for club in data["clubs"]:
            kg.sync_club(club["id"], club["name"])
    if "matches" in data:
        for m in data["matches"]:
            kg.sync_match(m["id"], m["home_id"], m["away_id"], m.get("date", ""))

    return jsonify({"status": "synced"})


if __name__ == "__main__":
    port = int(os.getenv("AI_AGENT_PORT", "5001"))
    app.run(host="0.0.0.0", port=port)
