FROM python:3.11-slim-bookworm

ENV PYTHONUNBUFFERED=1 \
    PYTHONDONTWRITEBYTECODE=1

WORKDIR /app

COPY requirements.txt ./

RUN pip install --no-cache-dir -r requirements.txt
RUN python -c "from fastembed import TextEmbedding; TextEmbedding(model_name='sentence-transformers/all-MiniLM-L6-v2')"

COPY ./app ./app
COPY seed_qdrant.py ./

EXPOSE 8122
CMD ["uvicorn", "app.main:app", "--host", "0.0.0.0", "--port", "8122"]