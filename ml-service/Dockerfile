FROM python:3.11-slim

WORKDIR /app

# LightGBM / XGBoost runtime (libgomp)
RUN apt-get update \
    && apt-get install -y --no-install-recommends libgomp1 \
    && rm -rf /var/lib/apt/lists/*

COPY ml-service/requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

COPY ml-service/app/ ./app/
COPY ml-service/models/ ./models/
COPY ml-pipeline/ ./ml-pipeline/
COPY ml-service/docker-entrypoint.sh /docker-entrypoint.sh
RUN sed -i 's/\r$//' /docker-entrypoint.sh && chmod +x /docker-entrypoint.sh

ENV ML_MODELS_DIR=/app/models

# Modèles demo intégrés à l'image (fallback si le volume hôte est vide)
RUN ML_MODELS_DIR=/app/models python /app/ml-pipeline/train_demo.py

EXPOSE 5000
ENTRYPOINT ["/docker-entrypoint.sh"]
