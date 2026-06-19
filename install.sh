#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
VERSION="0.1.0-dev"

usage() {
  cat <<EOF
gem2-lfs installer v${VERSION}

Usage: install.sh <mode> [options]

Modes:
  l0       Local binary install (sqlite-only, no Docker)
  docker   Docker Compose install (sqlite-only or sqlite-ollama)

Options:
  --port PORT          Server port (default: 9090)
  --db-path PATH       Database file path (default: .gem2-lfs/data.db)
  --mode MODE          sqlite-only or sqlite-ollama (default: sqlite-only)
  --ollama-url URL     Ollama API URL (default: http://localhost:11434)

Examples:
  install.sh l0
  install.sh l0 --mode sqlite-ollama
  install.sh docker
  install.sh docker --mode sqlite-ollama
EOF
  exit 1
}

install_l0() {
  echo "=== gem2-lfs L0 Install (local binary) ==="

  # Check Go.
  if ! command -v go &>/dev/null; then
    echo "ERROR: Go is required for local install. Install Go 1.25+ first."
    exit 1
  fi

  # Build.
  echo "Building gem2-lfs..."
  cd "${SCRIPT_DIR}"
  go build -o gem2-lfs ./cmd/gem2-lfs/
  echo "Built: ${SCRIPT_DIR}/gem2-lfs"

  # Initialize.
  echo "Initializing database..."
  ./gem2-lfs init --db-path "${DB_PATH}" --mode "${MODE}"

  # Doctor check.
  echo ""
  ./gem2-lfs doctor --db-path "${DB_PATH}" --ollama-url "${OLLAMA_URL}"

  echo ""
  echo "=== Install complete ==="
  echo "Start server: ./gem2-lfs serve --port ${PORT} --db-path ${DB_PATH} --mode ${MODE}"
}

install_docker() {
  echo "=== gem2-lfs Docker Install ==="

  # Check Docker.
  if ! command -v docker &>/dev/null; then
    echo "ERROR: Docker is required. Install Docker first."
    exit 1
  fi

  cd "${SCRIPT_DIR}"

  if [ "${MODE}" = "sqlite-ollama" ]; then
    echo "Building with Ollama support..."
    # Use docker compose with ollama service.
    docker compose -f docker-compose.yaml -f docker-compose.ollama.yaml up -d --build 2>/dev/null || \
      docker compose up -d --build
  else
    echo "Building sqlite-only..."
    docker compose up -d --build
  fi

  echo ""
  echo "=== Docker install complete ==="
  echo "Server running at http://localhost:${PORT}"
  echo "Check: curl http://localhost:${PORT}/capabilities"
}

# Parse arguments.
if [ $# -lt 1 ]; then
  usage
fi

INSTALL_MODE="$1"
shift

PORT="9090"
DB_PATH=".gem2-lfs/data.db"
MODE="sqlite-only"
OLLAMA_URL="http://localhost:11434"

while [ $# -gt 0 ]; do
  case "$1" in
    --port)    PORT="$2"; shift 2 ;;
    --db-path) DB_PATH="$2"; shift 2 ;;
    --mode)    MODE="$2"; shift 2 ;;
    --ollama-url) OLLAMA_URL="$2"; shift 2 ;;
    *)         echo "Unknown option: $1"; usage ;;
  esac
done

case "${INSTALL_MODE}" in
  l0)     install_l0 ;;
  docker) install_docker ;;
  *)      echo "Unknown mode: ${INSTALL_MODE}"; usage ;;
esac
