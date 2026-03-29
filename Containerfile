FROM python:3.11-slim

LABEL maintainer="Kevin Pirnie"
LABEL description="Recursive media library scanner and M3U playlist generator"

# libmediainfo is required by pymediainfo
RUN apt-get update && apt-get install -y --no-install-recommends \
    libmediainfo0v5 \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

COPY . .

# Conventional mount points
VOLUME ["/media", "/output"]

ENTRYPOINT ["python", "scanner.py"]
CMD ["--help"]
