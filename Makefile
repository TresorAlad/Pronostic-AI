.PHONY: up down infra train build test

up:
	docker compose up -d

down:
	docker compose down

infra:
	docker compose up -d postgres redis neo4j

train:
	cd ml-pipeline && ML_MODELS_DIR=../ml-service/models MLFLOW_TRACKING_URI=sqlite:////tmp/prono-mlruns/mlflow.db /tmp/prono-ml-venv/bin/python train.py

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

evaluate:
	cd ml-pipeline && /tmp/prono-ml-venv/bin/python evaluation/job_post_match.py

test-go:
	cd collector && go vet ./...
	cd backend && go vet ./...

test-python:
	cd ml-service && python -c "from app.predictor import ModelRegistry"
	cd ai-agent && python -c "from app.agent import analyze"
