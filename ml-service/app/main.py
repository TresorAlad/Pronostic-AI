"""Flask ML inference service."""

import os
import sys

from dotenv import load_dotenv
from flask import Flask, jsonify, request
from flask_cors import CORS

load_dotenv()

sys.path.insert(0, os.path.dirname(__file__))
from predictor import ModelRegistry

app = Flask(__name__)
CORS(app)

registry = ModelRegistry()


@app.route("/health", methods=["GET"])
def health():
    return jsonify({"status": "ok"})


@app.route("/models/status", methods=["GET"])
def models_status():
    return jsonify(registry.status())


@app.route("/predict/match", methods=["POST"])
def predict_match():
    data = request.get_json()
    if not data:
        return jsonify({"error": "JSON body required"}), 400

    features = data.get("features", data)
    match_id = data.get("match_id", "unknown")

    result = registry.predict_match(features, is_live=False)
    result["match_id"] = match_id
    return jsonify(result)


@app.route("/predict/match/live", methods=["POST"])
def predict_match_live():
    data = request.get_json()
    if not data:
        return jsonify({"error": "JSON body required"}), 400

    features = data.get("features", data)
    match_id = data.get("match_id", "unknown")

    result = registry.predict_match(features, is_live=True)
    result["match_id"] = match_id
    result["is_live"] = True
    return jsonify(result)


@app.route("/evaluate/match", methods=["POST"])
def evaluate_match():
    data = request.get_json()
    if not data:
        return jsonify({"error": "JSON body required"}), 400

    prediction_id = data.get("prediction_id")
    actual_outcomes = data.get("actual_outcomes", {})

    results = []
    for market, actual in actual_outcomes.items():
        predicted_prob = data.get("predictions", {}).get(market, 0.5)
        is_correct = (predicted_prob >= 0.5) == actual
        results.append({
            "market": market,
            "predicted_probability": predicted_prob,
            "actual_outcome": actual,
            "is_correct": is_correct,
        })

    return jsonify({
        "prediction_id": prediction_id,
        "outcomes": results,
        "evaluated_at": registry.predict_match({})["data_snapshot_at"],
    })


if __name__ == "__main__":
    port = int(os.getenv("ML_SERVICE_PORT", "5000"))
    app.run(host="0.0.0.0", port=port, debug=os.getenv("FLASK_DEBUG", "false") == "true")
