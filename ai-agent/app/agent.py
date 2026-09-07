"""LangGraph agent for match analysis - never invents probabilities."""

from __future__ import annotations

import json
import os
from typing import TypedDict

from langgraph.graph import END, StateGraph


class AgentState(TypedDict):
    match_id: str
    home_team: str
    away_team: str
    predictions: dict
    confidence: dict
    features: dict
    no_bet_recommended: bool
    context: dict
    analysis: str
    reasons: list
    recommended_markets: list
    abstain: bool


CONFIDENCE_THRESHOLD = float(os.getenv("AI_CONFIDENCE_THRESHOLD", "0.55"))


def fetch_context(state: AgentState) -> AgentState:
    """Gather contextual information from features."""
    features = state.get("features", {})
    context = {
        "home_form": features.get("home_form", 0),
        "away_form": features.get("away_form", 0),
        "home_xg": features.get("home_xg_avg_5"),
        "away_xg": features.get("away_xg_avg_5"),
        "home_possession": features.get("home_possession_avg_5"),
        "away_possession": features.get("away_possession_avg_5"),
        "home_availability": features.get("home_availability", 1.0),
        "away_availability": features.get("away_availability", 1.0),
    }
    state["context"] = context
    return state


def validate_ml_output(state: AgentState) -> AgentState:
    """Ensure we only use ML-provided probabilities."""
    predictions = state.get("predictions", {})
    if not predictions:
        state["abstain"] = True
        state["analysis"] = "Donnees ML insuffisantes pour produire une analyse."
        state["reasons"] = []
        state["recommended_markets"] = []
    return state


def _get_llm():
    """Instantiate LLM from env (Mistral by default)."""
    provider = os.getenv("LLM_PROVIDER", "mistral").lower()
    model = os.getenv("LLM_MODEL", "ministral-14b-2512")
    temperature = float(os.getenv("LLM_TEMPERATURE", "0.3"))

    if provider == "mistral":
        from langchain_mistralai import ChatMistralAI

        api_key = os.getenv("MISTRAL_API_KEY") or os.getenv("LLM_API_KEY")
        if not api_key:
            raise ValueError("MISTRAL_API_KEY manquante")
        return ChatMistralAI(
            model=model,
            api_key=api_key,
            endpoint=os.getenv("MISTRAL_API_BASE", "https://api.mistral.ai/v1"),
            temperature=temperature,
            max_tokens=1024,
        )

    from langchain_openai import ChatOpenAI

    return ChatOpenAI(model=model, temperature=temperature)


def analyze_match(state: AgentState) -> AgentState:
    """Generate analysis using LLM or rule-based fallback."""
    if state.get("abstain"):
        return state

    home = state["home_team"]
    away = state["away_team"]
    ctx = state.get("context", {})
    predictions = state["predictions"]
    confidence = state["confidence"]

    mistral_key = os.getenv("MISTRAL_API_KEY", "")
    openai_key = os.getenv("LLM_API_KEY", "")
    has_llm = (mistral_key and mistral_key != "your_mistral_api_key_here") or (
        openai_key and openai_key != "your_llm_api_key_here"
    )

    if has_llm:
        try:
            analysis, reasons = _llm_analyze(home, away, predictions, confidence, ctx)
            state["analysis"] = analysis
            state["reasons"] = reasons
        except Exception:
            analysis, reasons = _rule_based_analyze(home, away, predictions, confidence, ctx)
            state["analysis"] = analysis
            state["reasons"] = reasons
    else:
        analysis, reasons = _rule_based_analyze(home, away, predictions, confidence, ctx)
        state["analysis"] = analysis
        state["reasons"] = reasons

    return state


def recommend_or_abstain(state: AgentState) -> AgentState:
    """Select recommended markets or abstain."""
    if state.get("no_bet_recommended") or state.get("abstain"):
        state["abstain"] = True
        state["recommended_markets"] = []
        if not state.get("analysis"):
            state["analysis"] = "Aucun pronostic recommande. Confiance insuffisante."
        return state

    confidence = state.get("confidence", {})
    predictions = state.get("predictions", {})
    markets = []

    market_labels = {
        "home_win": "Victoire domicile",
        "draw": "Match nul",
        "away_win": "Victoire exterieur",
        "over_1_5": "Plus de 1.5 buts",
        "over_2_5": "Plus de 2.5 buts",
        "over_3_5": "Plus de 3.5 buts",
        "btts": "Les deux equipes marquent",
        "over_corners_9.5": "Plus de 9.5 corners",
        "over_shots_22_5": "Plus de 22.5 tirs",
        "over_shots_on_target_8_5": "Plus de 8.5 tirs cadres",
    }

    for market, conf in confidence.items():
        if conf >= CONFIDENCE_THRESHOLD:
            markets.append({
                "market": market,
                "label": market_labels.get(market, market),
                "confidence": conf,
                "probability": predictions.get(market, conf),
            })

    markets.sort(key=lambda x: x["confidence"], reverse=True)
    state["recommended_markets"] = markets[:3]

    if not markets:
        state["abstain"] = True
        state["analysis"] += "\n\nAucun marche ne depasse le seuil de confiance minimum."

    return state


def _rule_based_analyze(home, away, predictions, confidence, ctx) -> tuple[str, list]:
    """Fallback analysis without LLM."""
    parts = [f"Analyse du match {home} vs {away}.\n"]

    home_form = ctx.get("home_form", 0)
    away_form = ctx.get("away_form", 0)
    if home_form > away_form + 0.15:
        parts.append(f"{home} affiche une meilleure forme recente.")
    elif away_form > home_form + 0.15:
        parts.append(f"{away} est en meilleure dynamique.")

    if ctx.get("home_xg") and ctx.get("away_xg"):
        parts.append(
            f"xG moyen: {home} {ctx['home_xg']:.2f} vs {away} {ctx['away_xg']:.2f}."
        )

    home_win = predictions.get("home_win", 0)
    over_25 = predictions.get("over_2_5", 0)
    btts = predictions.get("btts", 0)

    parts.append(
        f"Probabilités du modele: victoire {home} {home_win:.0%}, "
        f"+2.5 buts {over_25:.0%}, BTTS {btts:.0%}."
    )

    reasons = []
    if over_25 > 0.6:
        reasons.append("Historique offensif favorable au over 2.5")
    if btts > 0.55:
        reasons.append("Les deux equipes marquent regulierement")
    if home_win > 0.5:
        reasons.append("Avantage domicile et forme superieure")
    if ctx.get("home_xg", 0) and ctx["home_xg"] > 1.5:
        reasons.append("xG eleve pour l'equipe domicile")
    if ctx.get("home_possession", 0) > 55:
        reasons.append("Domination habituelle au possession")

    return "\n".join(parts), reasons


def _llm_analyze(home, away, predictions, confidence, ctx) -> tuple[str, list]:
    """LLM-based analysis with strict probability grounding."""
    from langchain_core.messages import HumanMessage, SystemMessage

    llm = _get_llm()

    system_prompt = """Tu es un analyste football expert. REGLES STRICTES:
1. Tu ne dois JAMAIS inventer de probabilites. Utilise UNIQUEMENT celles fournies.
2. Cite les probabilites exactes du JSON ML dans ton analyse.
3. Si aucune probabilite ne depasse 55%, recommande de ne pas parier.
4. Reponds en francais, de maniere professionnelle.
5. Fournis une analyse en 2-3 paragraphes et une liste de raisons."""

    user_prompt = f"""Match: {home} vs {away}

Probabilites ML (NE PAS MODIFIER):
{json.dumps(predictions, indent=2)}

Confiance:
{json.dumps(confidence, indent=2)}

Contexte:
{json.dumps(ctx, indent=2)}

Produis:
1. ANALYSE: texte analytique
2. RAISONS: liste de 3-5 facteurs cles (prefixe chaque raison par "- ")"""

    response = llm.invoke([
        SystemMessage(content=system_prompt),
        HumanMessage(content=user_prompt),
    ])

    text = response.content if isinstance(response.content, str) else str(response.content)
    reasons = []
    analysis = text
    if "RAISONS:" in text:
        parts = text.split("RAISONS:")
        analysis = parts[0].replace("ANALYSE:", "").strip()
        for line in parts[1].strip().split("\n"):
            line = line.strip().lstrip("- ").strip()
            if line:
                reasons.append(line)

    return analysis, reasons


def build_graph() -> StateGraph:
    graph = StateGraph(AgentState)
    graph.add_node("fetch_context", fetch_context)
    graph.add_node("validate_ml_output", validate_ml_output)
    graph.add_node("analyze_match", analyze_match)
    graph.add_node("recommend_or_abstain", recommend_or_abstain)

    graph.set_entry_point("fetch_context")
    graph.add_edge("fetch_context", "validate_ml_output")
    graph.add_edge("validate_ml_output", "analyze_match")
    graph.add_edge("analyze_match", "recommend_or_abstain")
    graph.add_edge("recommend_or_abstain", END)

    return graph.compile()


agent = build_graph()


def analyze(payload: dict) -> dict:
    initial_state: AgentState = {
        "match_id": payload.get("match_id", ""),
        "home_team": payload.get("home_team", ""),
        "away_team": payload.get("away_team", ""),
        "predictions": payload.get("predictions", {}),
        "confidence": payload.get("confidence", {}),
        "features": payload.get("features", {}),
        "no_bet_recommended": payload.get("no_bet_recommended", False),
        "context": {},
        "analysis": "",
        "reasons": [],
        "recommended_markets": [],
        "abstain": False,
    }
    result = agent.invoke(initial_state)
    return {
        "analysis": result.get("analysis", ""),
        "reasons": result.get("reasons", []),
        "recommended_markets": result.get("recommended_markets", []),
        "abstain": result.get("abstain", False),
    }
