FROM python:3.12-slim

WORKDIR /app

# Copy SDK & Sidecar codebase
COPY sdk/python /app/sdk/python
COPY sidecar /app/sidecar

# Install dependencies
RUN pip install --no-cache-dir fastapi uvicorn httpx cryptography numpy requests

EXPOSE 8085

CMD ["python", "sidecar/proxy.py"]
