# Football AI Predictor

Plateforme intelligente d'analyse et de prédiction des matchs de football.

## Architecture

- **collector/** - Ingestion API-Football (Go)
- **backend/** - API REST, auth, WebSocket (Go)
- **ml-service/** - Service d'inference ML (Flask/Python)
- **ml-pipeline/** - Feature engineering et entraînement (Python)
- **ai-agent/** - Agent IA explicatif (LangGraph)
- **frontend/** - Interface React

## Prérequis

- Docker & Docker Compose
- Go 1.22+
- Python 3.11+
- Node.js 20+

## Démarrage rapide

```bash
cp .env.example .env
# Renseigner API_FOOTBALL_KEY et LLM_API_KEY dans .env

docker compose up -d postgres redis neo4j
docker compose up -d
```

Services :
- Frontend : http://localhost:3000
- Backend API : http://localhost:8080
- ML Service : http://localhost:5000
- AI Agent : http://localhost:5001
- Neo4j Browser : http://localhost:7474

## Backfill historique

```bash
cd collector
go run ./cmd/collector backfill --start-year 2018
```

## Entraînement ML

Le venv Python est sur `/tmp/prono-ml-venv` (partition sda2) pour eviter de saturer `/home`.

```bash
make venv    # premiere installation des dependances ML
make train   # entraine les 4 modeles dans ml-service/models/
```

## Principe fondamental

- Le **ML** calcule les probabilités (XGBoost/LightGBM)
- L'**agent IA** explique et justifie (ne invente jamais de probabilités)
- **Aucune cote bookmaker** n'est utilisée comme feature

## Compétitions V1

Premier League, La Liga, Serie A, Bundesliga, Ligue 1
