# laya-service

Local FastAPI wrapper around the Laya decision model, run on a Mac M1 and
reachable from the Pi 5 Spring Boot app over LAN. Acts as a fast pre-filter
in front of the Qwen3-4B sentiment call in `SentimentService`.

Uses `laya-coreml` (`aac6fef/laya-multilingual-coreml`, CPU+GPU, 1024-token
context) — an **unofficial third-party CoreML port** of Convai's Apache-2.0
`laya-multilingual` weights, not a Convai release. The ANE-only variant
(`laya-multilingual-coreml-ane`) is capped at 96 tokens and can't fit the
combined multi-article text this service classifies, so it isn't used here.

`laya_backend.py`'s response parsing (`result["answers"]["sentiment"]["value"]`
/ `["probabilities"]`) is based on published examples, not a primary-source
API reference (huggingface.co was unreachable from this environment while
writing this). **Verify the actual response shape against a real `/classify`
call before trusting it in shadow mode** — check `laya.log` (`logger.debug`
prints the raw response) if the endpoint errors on startup.

## Setup

```bash
cd laya-service
python3 -m venv venv
source venv/bin/activate
pip install -r requirements.txt
```

## First run (downloads weights, needs network)

```bash
export LAYA_API_KEY=<choose a secret, share it with the Pi 5 side's
  laya.service.api-key property>
./run.sh
```

The first request triggers a Hugging Face Hub download of the model weights
to `~/.cache/huggingface/hub`. Once that cache is populated, `run.sh`
automatically sets `HF_HUB_OFFLINE=1` on subsequent starts, so the service
runs without network access afterward.

Confirm it's up:

```bash
curl http://localhost:8000/health
curl -X POST http://localhost:8000/classify \
  -H "X-API-Key: $LAYA_API_KEY" -H "Content-Type: application/json" \
  -d '{"text": "Company X reports record quarterly profit, beats estimates."}'
```

## Keeping it running

Two options:

**launchd (recommended for always-on):**

1. Edit `com.swingtrade.laya.plist`, replacing `REPLACE_WITH_ABSOLUTE_PATH`
   with the absolute path to this directory and `REPLACE_WITH_API_KEY` with
   your chosen key.
2. `cp com.swingtrade.laya.plist ~/Library/LaunchAgents/`
3. `launchctl load ~/Library/LaunchAgents/com.swingtrade.laya.plist`
4. Logs land in `laya.log` / `laya.err.log` in this directory.
5. To stop: `launchctl unload ~/Library/LaunchAgents/com.swingtrade.laya.plist`

**Foreground / background process (for testing):**

```bash
./run.sh              # foreground
./run.sh &             # background, tied to the terminal session
```

## Network / firewall

The Pi 5 reaches this service at `http://<mac-lan-ip>:8000`. On the Mac,
allow incoming connections on port 8000 for Python/uvicorn in
System Settings → Network → Firewall if enabled. Find the Mac's LAN IP with
`ipconfig getifaddr en0` (or `en1` for Wi-Fi vs. Ethernet as applicable).
