#!/usr/bin/env bash
# Start MLX LLM server for SwingTrade sentiment analysis.
# Usage: ./infra/mlx-llm.sh [start|stop|status|restart]
#
# Defaults:
#   Model:  Qwen/Qwen2.5-3B-Instruct
#   Port:   8081
#   Host:   0.0.0.0
#   PID:    ~/.swingtrade/mlx.pid

set -euo pipefail

MODEL="${MLX_MODEL:-Qwen/Qwen2.5-3B-Instruct}"
PORT="${MLX_PORT:-8081}"
HOST="${MLX_HOST:-0.0.0.0}"
PID_DIR="$HOME/.swingtrade"
PID_FILE="$PID_DIR/mlx.pid"
LOG_FILE="$PID_DIR/mlx.log"

mkdir -p "$PID_DIR"

# ------------------------------------------------------------------
# Install mlx-lm if missing
# ------------------------------------------------------------------
install_mlx() {
  if python3 -c "import mlx_lm" 2>/dev/null; then
    echo "mlx-lm already installed"
    return
  fi

  echo "Installing mlx-lm ..."
  pip3 install mlx-lm
  echo "mlx-lm installed"
}

# ------------------------------------------------------------------
# Start
# ------------------------------------------------------------------
start() {
  install_mlx

  if is_running; then
    echo "MLX server already running (PID $(cat "$PID_FILE"))"
    return
  fi

  echo "Starting MLX server: model=$MODEL port=$PORT host=$HOST"
  nohup python3 -m mlx_lm.server \
    --model "$MODEL" \
    --port "$PORT" \
    --host "$HOST" \
    > "$LOG_FILE" 2>&1 &

  local pid=$!
  echo "$pid" > "$PID_FILE"
  echo "Started MLX server (PID $pid)"

  # Wait for health
  local waited=0
  while [ $waited -lt 120 ]; do
    if python3 -c "
import urllib.request, sys
try:
    urllib.request.urlopen('http://127.0.0.1:$PORT/health', timeout=2)
    sys.exit(0)
except:
    sys.exit(1)
" 2>/dev/null; then
      echo "MLX server healthy after ${waited}s"
      return
    fi
    sleep 1
    waited=$((waited + 1))
  done

  echo "WARNING: MLX server did not become healthy within 120s"
  echo "Check log: $LOG_FILE"
}

# ------------------------------------------------------------------
# Stop
# ------------------------------------------------------------------
stop() {
  if ! is_running; then
    echo "MLX server not running"
    return
  fi

  local pid
  pid=$(cat "$PID_FILE")
  echo "Stopping MLX server (PID $pid) ..."
  kill "$pid" 2>/dev/null || true
  # Also kill by port
  lsof -ti:"$PORT" | xargs kill -TERM 2>/dev/null || true
  rm -f "$PID_FILE"
  echo "Stopped"
}

# ------------------------------------------------------------------
# Status
# ------------------------------------------------------------------
status() {
  if is_running; then
    local pid
    pid=$(cat "$PID_FILE")
    echo "MLX server running (PID $pid)"
  else
    echo "MLX server not running"
  fi
}

# ------------------------------------------------------------------
# Helpers
# ------------------------------------------------------------------
is_running() {
  [ -f "$PID_FILE" ] || return 1
  local pid
  pid=$(cat "$PID_FILE")
  kill -0 "$pid" 2>/dev/null
}

# ------------------------------------------------------------------
# Main
# ------------------------------------------------------------------
case "${1:-start}" in
  start)   start   ;;
  stop)    stop    ;;
  status)  status  ;;
  restart) stop; start ;;
  *)
    echo "Usage: $0 {start|stop|status|restart}"
    exit 1
    ;;
esac