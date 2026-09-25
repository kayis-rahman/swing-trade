"""FastAPI wrapper exposing the Laya decision model as a local sentiment classifier.

Runs on the Mac M1, reachable from the Pi 5 Spring Boot app over LAN. Wraps the
unofficial laya-coreml port (Apple Neural Engine) so the swing-trade pipeline
can pre-filter news sentiment before falling back to the heavier Qwen3-4B call.
"""
import logging
import os
import time

from fastapi import FastAPI, Header, HTTPException
from pydantic import BaseModel

from laya_backend import LayaBackend

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger("laya-service")

API_KEY = os.environ.get("LAYA_API_KEY")
if not API_KEY:
    raise RuntimeError("LAYA_API_KEY must be set before starting the service")

app = FastAPI(title="laya-sentiment-service")
backend = LayaBackend()


class ClassifyRequest(BaseModel):
    text: str


class ClassifyResponse(BaseModel):
    sentiment: str
    confidence: float


def _check_api_key(x_api_key: str | None) -> None:
    if x_api_key != API_KEY:
        raise HTTPException(status_code=401, detail="invalid or missing API key")


@app.get("/health")
def health() -> dict:
    return {"status": "ok", "model_loaded": backend.is_loaded()}


@app.post("/classify", response_model=ClassifyResponse)
def classify(req: ClassifyRequest, x_api_key: str | None = Header(default=None)) -> ClassifyResponse:
    _check_api_key(x_api_key)
    if not req.text or not req.text.strip():
        raise HTTPException(status_code=400, detail="text must not be blank")

    start = time.monotonic()
    sentiment, confidence = backend.classify(req.text)
    latency_ms = (time.monotonic() - start) * 1000
    logger.info("classified text (%d chars) -> %s (%.2f) in %.1fms",
                len(req.text), sentiment, confidence, latency_ms)
    return ClassifyResponse(sentiment=sentiment, confidence=confidence)
