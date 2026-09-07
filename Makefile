.PHONY: up down infra train train-demo retrain backfill backfill-stats sync-stats evaluate sync-neo4j health build test

up:
	docker compose up -d --build

down:
	docker compose down

logs:
	docker compose logs -f

infra:
	docker compose up -d postgres redis neo4j

train:
	cd ml-pipeline && ML_MODELS_DIR=../ml-service/models MLFLOW_TRACKING_URI=sqlite:////tmp/prono-mlruns/mlflow.db /tmp/prono-ml-venv/bin/python train.py

train-demo:
	cd ml-pipeline && ML_MODELS_DIR=../ml-service/models /tmp/prono-ml-venv/bin/python train_demo.py

retrain:
	./scripts/retrain.sh

venv:
	python3 -m venv /tmp/prono-ml-venv
	/tmp/prono-ml-venv/bin/pip install -r ml-pipeline/requirements.txt

build-go:
	cd collector && go build ./cmd/collector
	cd backend && go build ./cmd/server

build-frontend:
	cd frontend && npm install && npm run build

backfill:
	cd collector && go run ./cmd/collector -mode=backfill -start-year=2018

sync-stats:
	cd collector && go run ./cmd/collector -mode=sync-stats -limit=500

backfill-stats:
	./scripts/backfill-stats.sh 10 500 45

sync-stats-all:
	cd collector && go run ./cmd/collector -mode=sync-stats-all -batches=10 -limit=500 -pause=45

evaluate:
	cd ml-pipeline && /tmp/prono-ml-venv/bin/python evaluation/job_post_match.py

sync-neo4j:
	/tmp/prono-ml-venv/bin/python scripts/sync-neo4j.py

health:
	./scripts/health-check.sh

test-go:
	cd collector && go vet ./... && go test ./...
	cd backend && go vet ./... && go test -race ./...

test-python:
	cd ml-pipeline && python -m pytest tests/ -q || true
	cd ml-service && python -c "from app.predictor import ModelRegistry"
	cd ai-agent && python -c "from app.agent import analyze"

test-frontend:
	cd frontend && npm run test
