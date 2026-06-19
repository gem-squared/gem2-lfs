# @gem_squared/gem2-lfs

Lightweight Local Knowledge Store — a SQLite-backed drop-in replacement for gem2-kg.

## Install

```bash
npm install -g @gem_squared/gem2-lfs
```

Only the binary for your current platform is downloaded (~15 MB). Supported platforms:

| OS | Arch | Package |
|----|------|---------|
| macOS | Apple Silicon | `@gem_squared/gem2-lfs-darwin-arm64` |
| macOS | Intel | `@gem_squared/gem2-lfs-darwin-x64` |
| Linux | x86_64 | `@gem_squared/gem2-lfs-linux-x64` |
| Linux | ARM64 | `@gem_squared/gem2-lfs-linux-arm64` |

## First-Time Setup

```bash
gem2-lfs setup
```

Interactive wizard that:
1. Initializes the SQLite database
2. Checks Ollama availability (for semantic search)
3. Registers gem2-lfs as an MCP server in `~/.claude.json`

## Usage

```bash
# Start the server
gem2-lfs serve --port 9090

# Or via npx (no global install)
npx @gem_squared/gem2-lfs serve --port 9090

# Initialize database
gem2-lfs init --db-path .gem2-lfs/data.db

# Health check
gem2-lfs doctor

# Print version
gem2-lfs version
```

## Modes

- **sqlite-only** (default): Local SQLite + FTS5 full-text search
- **sqlite-ollama**: SQLite + semantic search via Ollama embeddings

```bash
gem2-lfs serve --mode sqlite-ollama --ollama-url http://localhost:11434
```

## License

MIT
