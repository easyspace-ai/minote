# MarkItDown HTTP API — pairs with repo root docker-compose / Notex MARKITDOWN_URL.
# Self-contained build (PyPI markitdown); no submodule or vendored upstream tree required.
FROM python:3.13-slim-bookworm

ENV DEBIAN_FRONTEND=noninteractive
ENV EXIFTOOL_PATH=/usr/bin/exiftool
ENV FFMPEG_PATH=/usr/bin/ffmpeg

RUN apt-get update && apt-get install -y --no-install-recommends \
    curl \
    ffmpeg \
    exiftool \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app
COPY http_server.py /app/http_server.py

RUN pip --no-cache-dir install \
    "fastapi>=0.115.0" \
    "uvicorn[standard]>=0.32.0" \
    "python-multipart>=0.0.9" \
    && pip --no-cache-dir install "markitdown[all]"

EXPOSE 8787
ENV MARKITDOWN_ENABLE_PLUGINS=false

HEALTHCHECK --interval=30s --timeout=15s --start-period=90s --retries=3 \
    CMD curl -fsS http://127.0.0.1:8787/health || exit 1

CMD ["uvicorn", "http_server:app", "--host", "0.0.0.0", "--port", "8787"]
