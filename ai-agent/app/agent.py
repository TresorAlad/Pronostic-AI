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
    home_team_id: str
    away_team_id: str
    predictions: dict
    confidence: dict
    features: dict
    no_bet_recommended: bool
    context: dict
    odds: list
    analysis: str
    reasons: list
    recommended_markets: list
    abstain: bool


CONFIDENCE_THRESHOLD = float(os.getenv("AI_CONFIDENCE_THRESHOLD", "0.55"))


def fetch_context(state: AgentState) -> AgentState:
    """Gather contextual information from features, historique individuel et H2H."""
    features = state.get("features", {})
    context = {
        "home_form": features.get("home_form", 0),
        "away_form": features.get("away_form", 0),
        "home_xg": features.get("home_xg_avg_5"),
        "away_xg": features.get("away_xg_avg_5"),
        "home_possession": features.get("home_possession_avg_5"),
        "away_possession": features.get("away_possession_avg_5"),
        "home_fouls": features.get("home_fouls_avg_5"),
        "away_fouls": features.get("away_fouls_avg_5"),
        "home_cards": features.get("home_yellow_cards_avg_5"),
        "away_cards": features.get("away_yellow_cards_avg_5"),
        "home_offsides": features.get("home_offsides_avg_5"),
        "away_offsides": features.get("away_offsides_avg_5"),
        "home_availability": features.get("home_availability", 1.0),
        "away_availability": features.get("away_availability", 1.0),
    }

    home_history = features.get("home_recent_matches") or []
    away_history = features.get("away_recent_matches") or []
    h2h_db = features.get("h2h_history") or []
    if home_history:
        context["home_recent_matches"] = home_history[:10]
        context["home_competitions"] = sorted(
            {m.get("league_name") for m in home_history if m.get("league_name")}
        )
    if away_history:
        context["away_recent_matches"] = away_history[:10]
        context["away_competitions"] = sorted(
            {m.get("league_name") for m in away_history if m.get("league_name")}
        )
    if h2h_db:
        context["h2h_history"] = h2h_db[:8]
        context["h2h_matches"] = len(h2h_db)

    if features.get("data_quality"):
        context["data_quality"] = features.get("data_quality")
    if features.get("history_source_home"):
        context["history_source_home"] = features.get("history_source_home")
    if features.get("history_source_away"):
        context["history_source_away"] = features.get("history_source_away")

    home_id = features.get("home_team_id") or state.get("home_team_id")
    away_id = features.get("away_team_id") or state.get("away_team_id")
    if home_id and away_id:
        try:
            from app.neo4j_sync import KnowledgeGraph

            kg = KnowledgeGraph()
            h2h = kg.get_h2h_context(str(home_id), str(away_id), limit=5)
            kg.close()
            if h2h:
                context["h2h_graph_recent"] = h2h[:3]
            if "h2h_matches" not in context:
                context["h2h_matches"] = len(h2h)
        except Exception:
            if "h2h_matches" not in context:
                context["h2h_matches"] = 0

    odds = state.get("odds") or []
    if odds:
        context["bookmaker_odds"] = odds
        value_rows = [
            o for o in odds
            if isinstance(o, dict) and float(o.get("value_edge") or 0) > 0.05
        ]
        if value_rows:
            context["value_bets"] = value_rows[:5]

    state["context"] = context
    return state


def validate_ml_output(state: AgentState) -> AgentState:
    """Ensure we only use ML-provided probabilities when available."""
    predictions = state.get("predictions", {})
    features = state.get("features", {})
    has_team_history = bool(
        features.get("home_recent_matches") or features.get("away_recent_matches")
    )
    if not predictions and not has_team_history:
        state["abstain"] = True
        state["analysis"] = "Donnees ML et historique equipes insuffisants pour produire une analyse."
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
    predictions = state.get("predictions") or {}
    confidence = state.get("confidence") or {}

    mistral_key = os.getenv("MISTRAL_API_KEY", "")
    openai_key = os.getenv("LLM_API_KEY", "")
    has_llm = (mistral_key and mistral_key != "your_mistral_api_key_here") or (
        openai_key and openai_key != "your_llm_api_key_here"
    )

    if has_llm and predictions:
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

    if not state.get("reasons"):
        _, reasons = _rule_based_analyze(home, away, predictions, confidence, ctx)
        state["reasons"] = reasons

    return state


def recommend_or_abstain(state: AgentState) -> AgentState:
    """Select recommended markets or abstain."""
    if state.get("abstain"):
        state["recommended_markets"] = []
        if not state.get("analysis"):
            state["analysis"] = "Données ML insuffisantes pour produire une analyse."
        return state

    confidence = state.get("confidence", {})
    predictions = state.get("predictions", {})
    markets = []

    market_labels = {
        "home_win": "Victoire domicile",
        "draw": "Match nul",
        "away_win": "Victoire extérieur",
        "over_1_5": "Plus de 1,5 buts",
        "over_2_5": "Plus de 2,5 buts",
        "over_3_5": "Plus de 3,5 buts",
        "under_1_5": "Moins de 1,5 buts",
        "under_2_5": "Moins de 2,5 buts",
        "btts": "Les deux équipes marquent",
        "btts_no": "Les deux équipes ne marquent pas",
        "over_corners_9_5": "Plus de 9,5 corners",
        "over_shots_22_5": "Plus de 22,5 tirs",
        "over_shots_on_target_8_5": "Plus de 8,5 tirs cadrés",
        "over_fouls_20_5": "Plus de 20,5 fautes",
        "over_fouls_22_5": "Plus de 22,5 fautes",
        "over_fouls_25_5": "Plus de 25,5 fautes",
        "over_cards_3_5": "Plus de 3,5 cartons",
        "over_cards_4_5": "Plus de 4,5 cartons",
        "over_cards_5_5": "Plus de 5,5 cartons",
        "over_offsides_2_5": "Plus de 2,5 hors-jeu",
        "over_offsides_3_5": "Plus de 3,5 hors-jeu",
        "over_offsides_4_5": "Plus de 4,5 hors-jeu",
        "home_possession_over_50": "Domicile > 50 % possession",
        "away_possession_over_50": "Extérieur > 50 % possession",
    }

    for market, conf in confidence.items():
        if market.startswith("predicted_") or market.startswith("double_chance_"):
            continue
        if conf >= CONFIDENCE_THRESHOLD:
            markets.append({
                "market": market,
                "label": market_labels.get(market, market),
                "confidence": conf,
                "probability": predictions.get(market, conf),
            })

    markets.sort(key=lambda x: x["confidence"], reverse=True)
    state["recommended_markets"] = markets[:5]

    if not markets and not state.get("no_bet_recommended"):
        state["abstain"] = False
    elif not markets:
        state["abstain"] = True
        if state.get("analysis"):
            state["analysis"] += "\n\nAucun marché ne dépasse le seuil de confiance minimum."

    return state


def _format_recent_matches(team_name: str, matches: list) -> str:
    if not matches:
        return ""
    lines = [f"Derniers matchs de {team_name} (championnat, coupes, toutes compétitions) :"]
    for m in matches[:6]:
        venue = "dom." if m.get("is_home") else "ext."
        comp = m.get("league_name", "?")
        opp = m.get("opponent", "?")
        gf = m.get("goals_for", 0)
        ga = m.get("goals_against", 0)
        res = m.get("result", "?")
        date = m.get("date", "")
        lines.append(f"- {date} [{comp}] {venue} vs {opp} {gf}-{ga} ({res})")
    return "\n".join(lines)


def _format_h2h(h2h: list) -> str:
    if not h2h:
        return ""
    lines = ["Confrontations directes (toutes compétitions) :"]
    for m in h2h[:5]:
        comp = m.get("league_name", "?")
        date = m.get("date", "")
        home = m.get("home_team", "?")
        away = m.get("away_team", "?")
        hs = m.get("home_score", 0)
        aws = m.get("away_score", 0)
        lines.append(f"- {date} [{comp}] {home} {hs}-{aws} {away}")
    return "\n".join(lines)


def _rule_based_analyze(home, away, predictions, confidence, ctx) -> tuple[str, list]:
    """Fallback analysis without LLM — s'appuie sur historique equipes et proba ML si dispo."""
    parts = [f"Synthese {home} vs {away}.\n"]
    reasons = []

    home_form = ctx.get("home_form", 0)
    away_form = ctx.get("away_form", 0)
    if home_form > away_form + 0.15:
        parts.append(f"{home} affiche une meilleure forme recente (toutes competitions).")
        reasons.append(f"Forme superieure de {home} sur les derniers matchs")
    elif away_form > home_form + 0.15:
        parts.append(f"{away} est en meilleure dynamique recente.")
        reasons.append(f"Dynamique favorable a {away}")

    home_hist = ctx.get("home_recent_matches") or []
    away_hist = ctx.get("away_recent_matches") or []
    if home_hist or away_hist:
        if home_hist:
            parts.append(_format_recent_matches(home, home_hist))
            wins = sum(1 for m in home_hist if m.get("result") == "W")
            reasons.append(f"{home} : {wins} victoire(s) sur {len(home_hist)} derniers matchs")
        if away_hist:
            parts.append(_format_recent_matches(away, away_hist))
            wins = sum(1 for m in away_hist if m.get("result") == "W")
            reasons.append(f"{away} : {wins} victoire(s) sur {len(away_hist)} derniers matchs")

    h2h_history = ctx.get("h2h_history") or []
    if h2h_history:
        parts.append(_format_h2h(h2h_history))
        reasons.append(f"{len(h2h_history)} confrontation(s) directe(s) analysee(s)")

    quality = ctx.get("data_quality", "")
    if quality in ("low", "estimated"):
        parts.append("Donnees partielles : estimation basee sur historique disponible et moyennes de ligue.")

    if predictions:
        home_win = predictions.get("home_win", 0)
        over_25 = predictions.get("over_2_5", 0)
        btts = predictions.get("btts", 0)
        parts.append(
            f"Probabilités du modele : victoire {home} {home_win:.0%}, "
            f"+2,5 buts {over_25:.0%}, les deux marquent {btts:.0%}."
        )
        if over_25 > 0.6:
            reasons.append("Historique offensif favorable au over 2,5 buts")
        if btts > 0.55:
            reasons.append("Les deux equipes marquent regulierement")
        if home_win > 0.5:
            reasons.append("Avantage domicile et forme superieure")

    if ctx.get("value_bets"):
        for row in ctx["value_bets"][:2]:
            ml_market = row.get("ml_market", row.get("market", ""))
            edge = float(row.get("value_edge") or 0)
            reasons.append(f"Value detecte sur {ml_market} (+{edge:.0%})")

    if not reasons:
        reasons.append("Analyse basee sur l'historique recent individuel de chaque club")

    return "\n".join(parts), reasons[:6]


def _llm_analyze(home, away, predictions, confidence, ctx) -> tuple[str, list]:
    """LLM-based analysis with strict probability grounding."""
    from langchain_core.messages import HumanMessage, SystemMessage

    llm = _get_llm()

    system_prompt = """Tu es un analyste football expert. RÈGLES STRICTES:
1. Tu ne dois JAMAIS inventer de probabilités. Utilise UNIQUEMENT celles fournies.
2. Cite les probabilités exactes du JSON ML dans ton analyse.
3. Couvre toutes les catégories disponibles : résultat (1X2), buts, tirs, corners, cartons, fautes, hors-jeu, possession.
4. Intègre l'historique individuel de chaque club (championnat, coupes, autres compétitions) ET les confrontations directes H2H.
5. Ne te limite pas au seul H2H : la forme récente toutes compétitions compte autant.
6. Si aucune probabilité ne dépasse 55 %, indique-le clairement sans inventer de picks.
7. Réponds en français, de manière professionnelle.
8. Fournis une analyse en 3-4 paragraphes structurés par thème et une liste de raisons.
9. Si des cotes bookmaker sont fournies, compare-les aux probabilités ML sans les modifier."""

    user_prompt = f"""Match: {home} vs {away}

Probabilités ML (NE PAS MODIFIER):
{json.dumps(predictions, indent=2)}

Confiance:
{json.dumps(confidence, indent=2)}

Contexte (moyennes récentes + historique individuel toutes compétitions + H2H + cotes):
{json.dumps(ctx, indent=2)}

Produis:
1. ANALYSE: texte couvrant victoire/résultat, buts, tirs, corners, cartons, fautes, hors-jeu et possession
2. RAISONS: liste de 4-6 facteurs clés (préfixe chaque raison par "- ")"""

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
        "home_team_id": payload.get("home_team_id", ""),
        "away_team_id": payload.get("away_team_id", ""),
        "predictions": payload.get("predictions", {}),
        "confidence": payload.get("confidence", {}),
        "features": payload.get("features", {}),
        "no_bet_recommended": payload.get("no_bet_recommended", False),
        "odds": payload.get("odds", []),
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
