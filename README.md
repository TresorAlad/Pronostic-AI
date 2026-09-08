# Football AI Predictor (Pronostic-AI)

Plateforme intelligente d'analyse et de prédiction des matchs de football.

## Architecture

- **collector/** - Ingestion API-Football (Go)
- **backend/** - API REST, auth, WebSocket (Go)
- **ml-service/** - Service d'inférence ML (Flask/Python)
- **ml-pipeline/** - Feature engineering et entraînement (Python)
- **ai-agent/** - Agent IA explicatif (LangGraph)
- **frontend/** - Interface React

## Prérequis

- Docker et Docker Compose
- Go 1.22+ (mode dev local uniquement)
- Python 3.11+ (mode dev local uniquement)
- Node.js 20+ (mode dev local uniquement)

## Démarrage en une commande (Docker Compose)

Toute la stack est définie dans [`docker-compose.yml`](docker-compose.yml) : PostgreSQL, Redis, Neo4j, backend, collector, ML, agent IA et frontend.

```bash
# 1. Configuration (une seule fois)
cp .env.example .env
# Renseigner au minimum API_FOOTBALL_KEY et JWT_SECRET dans .env

# 2. Lancer tout le projet
docker compose up -d --build
```

Équivalent via Makefile :

```bash
make up
```

Services disponibles après démarrage :

| Service | URL |
|---------|-----|
| Application (frontend) | http://localhost:5173 |
| Backend API | http://localhost:8080 |
| ML Service | http://localhost:5000 |
| AI Agent | http://localhost:5001 |
| Collector | http://localhost:8090 |
| Neo4j Browser | http://localhost:7474 |

Commandes utiles :

```bash
make logs    # suivre les logs de tous les conteneurs
make down    # arrêter et retirer la stack
make health  # vérifier que les services répondent
```

Les migrations PostgreSQL (`infra/postgres/migrations/`) sont appliquées automatiquement au premier démarrage du conteneur `postgres`.

## Deux modes de démarrage

| Mode | Commande | Usage |
|------|----------|-------|
| **Docker (recommandé)** | `docker compose up -d --build` ou `make up` | Stack complète via `docker-compose.yml`, backend `:8080`, frontend `:5173` |
| **Dev hot-reload** | `./scripts/dev-start.sh` | Backend `:8082`, Vite `:5173`, services locaux |

Ports harmonisés dans `.env.example` :
- Dev : `BACKEND_PORT=8082`, `VITE_API_URL=/api/v1` (proxy Vite)
- Docker : `BACKEND_PORT=8080`, nginx reverse proxy

## Base de données : local vs Neon

| Environnement | `DATABASE_URL` | Usage |
|---------------|----------------|-------|
| **Local dev** | `postgres://prono:prono_secret@localhost:5432/prono?sslmode=disable` | Docker Compose, tests |
| **Neon (prod/staging)** | URL `postgresql://...@...neon.tech/...` | Données partagées, prod |

Bascule : modifiez uniquement `DATABASE_URL` dans `.env`, puis relancez collector et backend.

```bash
# Local
make infra
export DATABASE_URL=postgres://prono:prono_secret@localhost:5432/prono?sslmode=disable

# Neon (exemple)
export DATABASE_URL=postgresql://user:pass@ep-xxx.neon.tech/neondb?sslmode=require
```

Appliquez les migrations sur les deux environnements :

```bash
psql "$DATABASE_URL" -f infra/postgres/migrations/001_initial_schema.sql
psql "$DATABASE_URL" -f infra/postgres/migrations/002_injuries_unique.sql
```

## Démarrage rapide (dev)

```bash
cp .env.example .env
# Renseigner API_FOOTBALL_KEY, DATABASE_URL, JWT_SECRET

make infra
./scripts/dev-start.sh
```

Services (ports dev par défaut) :
- Frontend : http://localhost:5173
- Backend API : http://localhost:8082 (ou `BACKEND_PORT`)
- ML Service : http://localhost:5002
- AI Agent : http://localhost:5001
- Neo4j Browser : http://localhost:7474

## Données (Phase 0)

```bash
# Historique Top 5 (2018 -> aujourd'hui)
make backfill

# Stats détaillées (cartons, fautes, corners...) - un batch
make sync-stats

# Rattrapage massif par batches (pause quota API)
make backfill-stats
# ou: ./scripts/backfill-stats.sh 10 500 45

# Mode collector intégré
cd collector && go run ./cmd/collector -mode=sync-stats-all -batches=10 -limit=500 -pause=45
```

Le collector daemon appelle automatiquement le pre-warm backend après `SyncToday` si `BACKEND_URL` est défini.

## Entraînement ML

Deux commandes distinctes :

| Commande | Script | Quand l'utiliser |
|----------|--------|------------------|
| `make train` | `ml-pipeline/train.py` | **Production** : entraîne sur Neon (`DATABASE_URL` requis) |
| `make train-demo` | `ml-pipeline/train_demo.py` | **CI/dev sans DB** : modèles synthétiques pour tester la stack |

```bash
make venv
make train          # prod - données réelles Neon
# Produit: ml-service/models/*.joblib
make retrain        # reload des modèles dans ml-service
```

Vérification : `GET http://localhost:5002/models/status` (4 modèles, version non-demo).

La CI hebdo (`.github/workflows/weekly-retrain.yml`) utilise `train.py` si `DATABASE_URL` est configuré, sinon `train_demo.py`.

## Évaluation

Le job Python délègue au backend (même logique que `store.go` : marchés score + stats).

```bash
make evaluate
curl -X POST http://localhost:8082/api/v1/evaluation/run
```

Le collector déclenche aussi l'évaluation après chaque `sync-live` (matchs terminés).

## Neo4j

```bash
make sync-neo4j   # après backfill pour alimenter le graphe H2H
```

Le widget santé frontend affiche l'état Neo4j via `/health` de l'agent IA.

## Santé stack

```bash
./scripts/health-check.sh
```

## Tests

```bash
make test-go        # auth, rate limit, coupons
make test-python    # job_post_match, dataset
make test-frontend  # hooks auth, marketLabels
```

## Principe fondamental

- Le **ML** calcule les probabilités (LightGBM/XGBoost)
- L'**agent IA** explique et justifie (ne invente jamais de probabilités)
- **Aucune cote bookmaker** n'est utilisée comme feature

## Compétitions V1

Premier League, La Liga, Serie A, Bundesliga, Ligue 1
