FROM python:3.11-slim
WORKDIR /app
COPY ml-service/requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt
COPY ml-service/app/ ./app/
COPY ml-service/models/ ./models/
COPY ml-pipeline/ ./ml-pipeline/
ENV ML_MODELS_DIR=/app/models
EXPOSE 5000
CMD ["gunicorn", "--bind", "0.0.0.0:5000", "--workers", "2", "app.main:app"]
