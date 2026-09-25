import importlib.util
from pathlib import Path


class FakeAgent:
    def __init__(self):
        self.calls = []

    def predict(self, text, schema):
        self.calls.append((text, schema))
        return {
            "answers": {
                "sentiment": {
                    "choice": "POSITIVE",
                    "probabilities": {"POSITIVE": 0.91},
                }
            }
        }


def load_backend_module():
    fake_laya = type("FakeLaya", (), {})()
    fake_laya.load = lambda _model_id, **_kwargs: None
    import sys

    sys.modules["laya_coreml"] = fake_laya
    spec = importlib.util.spec_from_file_location(
        "laya_backend_under_test", Path(__file__).with_name("laya_backend.py")
    )
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


def test_classify_uses_laya_schema_and_value_response():
    module = load_backend_module()
    agent = FakeAgent()
    backend = object.__new__(module.LayaBackend)
    backend._agent = agent

    sentiment, confidence = backend.classify("record profit")

    assert (sentiment, confidence) == ("POSITIVE", 0.91)
    text, schema = agent.calls[0]
    assert text == "record profit"
    assert schema["sentiment"]["type"] == "choice"
    assert schema["sentiment"]["criteria"] == ["POSITIVE", "NEUTRAL", "NEGATIVE"]
