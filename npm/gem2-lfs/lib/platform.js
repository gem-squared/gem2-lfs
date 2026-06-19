"use strict";

const os = require("os");
const path = require("path");

const fs = require("fs");

/**
 * Map Node.js platform/arch to npm package name suffix
 * and the local directory name (for dev/monorepo fallback).
 * Node.js uses "x64" for amd64 and "arm64" for aarch64.
 */
const PLATFORMS = {
  "darwin-arm64": { pkg: "@gem_squared/gem2-lfs-darwin-arm64", dir: "gem2-lfs-darwin-arm64" },
  "darwin-x64":   { pkg: "@gem_squared/gem2-lfs-darwin-x64",   dir: "gem2-lfs-darwin-x64" },
  "linux-x64":    { pkg: "@gem_squared/gem2-lfs-linux-x64",    dir: "gem2-lfs-linux-x64" },
  "linux-arm64":  { pkg: "@gem_squared/gem2-lfs-linux-arm64",  dir: "gem2-lfs-linux-arm64" },
};

function platformKey() {
  return `${os.platform()}-${os.arch()}`;
}

function getBinaryPath() {
  const key = platformKey();
  const entry = PLATFORMS[key];

  if (!entry) {
    throw new Error(
      `Unsupported platform: ${key}. ` +
        `gem2-lfs supports: ${Object.keys(PLATFORMS).join(", ")}`
    );
  }

  // 1. Try require.resolve (works when installed via npm)
  try {
    const pkgJson = require.resolve(`${entry.pkg}/package.json`);
    return path.join(path.dirname(pkgJson), "bin", "gem2-lfs");
  } catch (_) {
    // Fall through to dev fallback
  }

  // 2. Dev fallback: sibling directory (monorepo / repo checkout)
  const devPath = path.join(__dirname, "..", "..", entry.dir, "bin", "gem2-lfs");
  if (fs.existsSync(devPath)) {
    return devPath;
  }

  throw new Error(
    `Platform package ${entry.pkg} is not installed. ` +
      `Try reinstalling: npm install @gem_squared/gem2-lfs`
  );
}

module.exports = { getBinaryPath, PLATFORMS, platformKey };
