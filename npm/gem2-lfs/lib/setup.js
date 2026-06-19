"use strict";

const { execFileSync } = require("child_process");
const readline = require("readline");
const path = require("path");
const fs = require("fs");
const os = require("os");
const { getBinaryPath } = require("./platform");

function createLineReader() {
  const lines = [];
  const waiters = [];
  const rl = readline.createInterface({ input: process.stdin, output: process.stdout });

  rl.on("line", (line) => {
    if (waiters.length > 0) {
      waiters.shift()(line.trim());
    } else {
      lines.push(line.trim());
    }
  });

  rl.on("close", () => {
    while (waiters.length > 0) waiters.shift()("");
  });

  function ask(prompt) {
    process.stdout.write(prompt);
    if (lines.length > 0) {
      const line = lines.shift();
      process.stdout.write(line + "\n");
      return Promise.resolve(line);
    }
    return new Promise((resolve) => waiters.push(resolve));
  }

  return { ask, close: () => rl.close() };
}

async function runSetup() {
  const binary = getBinaryPath();
  let ver = "0.1.0";
  try {
    ver = execFileSync(binary, ["version"], { stdio: "pipe" }).toString().trim();
    ver = ver.replace(/^gem2-lfs\s+/, "");
  } catch (_) {}

  const lr = createLineReader();

  console.log();
  console.log(`gem2-lfs ${ver} — first-time setup`);
  console.log("─".repeat(36));

  let dbPath = ".gem2-lfs/data.db";

  // Phase 1: Database init
  try {
    console.log();
    console.log("[1/3] Database");
    const answer = await lr.ask(`      Path [${dbPath}]: `);
    if (answer) dbPath = answer;
    execFileSync(binary, ["init", "--db-path", dbPath], { stdio: "inherit" });
    console.log(`      ✓ Initialized at ${dbPath}`);
  } catch (e) {
    console.error(`      ✗ Database init failed: ${e.message}`);
  }

  // Phase 2: Ollama check
  let mode = "sqlite-only";
  try {
    console.log();
    console.log("[2/3] Ollama (optional — enables semantic search)");
    process.stdout.write("      Checking localhost:11434... ");
    const out = execFileSync(binary, ["doctor", "--ollama-url", "http://localhost:11434"], {
      stdio: "pipe",
    }).toString();

    if (out.includes("Ollama: OK")) {
      console.log("found.");
      console.log("      ✓ Ollama available. Mode: sqlite-ollama");
      mode = "sqlite-ollama";
    } else {
      console.log("not found.");
      console.log("      → Skip for now. Mode: sqlite-only (default).");
      console.log("      → To enable later: ollama pull nomic-embed-text:v1.5");
    }
  } catch (_) {
    console.log("not found.");
    console.log("      → Skip for now. Mode: sqlite-only (default).");
    console.log("      → To enable later: ollama pull nomic-embed-text:v1.5");
  }

  // Phase 3: MCP registration
  try {
    console.log();
    console.log("[3/3] Register as MCP server for Claude Code?");
    const answer = await lr.ask("      [Y/n]: ");
    const yes = !answer || answer.toLowerCase() === "y" || answer.toLowerCase() === "yes";

    if (yes) {
      const claudeConfigPath = path.join(os.homedir(), ".claude.json");
      let config = { mcpServers: {} };
      try {
        const raw = fs.readFileSync(claudeConfigPath, "utf8");
        config = JSON.parse(raw);
        if (!config.mcpServers) config.mcpServers = {};
      } catch (_) {}

      config.mcpServers["gem2-lfs"] = {
        command: "gem2-lfs",
        args: ["mcp", "--db-path", dbPath, "--mode", mode],
      };

      fs.writeFileSync(claudeConfigPath, JSON.stringify(config, null, 2) + "\n");
      console.log("      ✓ Added gem2-lfs to ~/.claude.json");
    } else {
      console.log("      → Skipped. Register manually: claude mcp add gem2-lfs -- gem2-lfs mcp");
    }
  } catch (e) {
    console.error(`      ✗ MCP registration failed: ${e.message}`);
  }

  lr.close();
  console.log();
  console.log(`Done. Start with: gem2-lfs serve --mode ${mode}`);
}

module.exports = { runSetup };
