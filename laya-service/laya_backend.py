"""Wraps the laya-coreml decision model for single-shot sentiment classification.

Uses the multilingual CoreML checkpoint (1024-token context, CPU+GPU) rather
than the ANE variant, which is capped at 96 tokens -- too small for the
combined multi-article news text this service is fed.
"""
import logging

import laya_coreml as laya

logger = logging.getLogger("laya-service")

MODEL_ID = "aac6fef/laya-multilingual-coreml"

# Rough chars-per-token budget for the model's 1024-token context, leaving
# headroom for the fixed question/schema overhead in the same call.
MAX_INPUT_CHARS = 3200

_SENTIMENT_SCHEMA = {
    "sentiment": {
        "type": "choice",
        "instructions": (
            "Classify the overall market sentiment of this news text for the "
            "stock it concerns."
        ),
        "options": ["POSITIVE", "NEUTRAL", "NEGATIVE"],
    }
}


class LayaBackend:
    def __init__(self) -> None:
        logger.info("loading %s", MODEL_ID)
        self._agent = laya.load(MODEL_ID)
        logger.info("model loaded")

    def is_loaded(self) -> bool:
        return self._agent is not None

    def classify(self, text: str) -> tuple[str, float]:
        trimmed = text if len(text) <= MAX_INPUT_CHARS else text[:MAX_INPUT_CHARS]
        result = self._agent.predict(trimmed, _SENTIMENT_SCHEMA)
        logger.debug("raw laya response: %r", result)
        answer = result["answers"]["sentiment"]
        sentiment = answer["value"]
        confidence = float(answer["probabilities"][sentiment])
        return sentiment, confidence
