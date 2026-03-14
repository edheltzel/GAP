#!/usr/bin/env node

import { spawn, execSync } from "node:child_process";
import { existsSync } from "node:fs";
import { platform } from "node:os";
import puppeteer from "puppeteer-core";

const useProfile = process.argv[2] === "--profile";

if (process.argv[2] && process.argv[2] !== "--profile") {
	console.log("Usage: browser-start.js [--profile]");
	console.log("\nOptions:");
	console.log("  --profile  Copy your default Chrome profile (cookies, logins)");
	process.exit(1);
}

// Find Chrome/Chromium binary
function findBrowser() {
	if (platform() === "darwin") {
		const paths = [
			"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
			"/Applications/Chromium.app/Contents/MacOS/Chromium",
		];
		for (const p of paths) if (existsSync(p)) return { path: p, profileDir: `${process.env.HOME}/Library/Application Support/Google/Chrome/` };
	} else {
		// Linux
		const names = ["google-chrome-stable", "google-chrome", "chromium-browser", "chromium"];
		for (const name of names) {
			try {
				const p = execSync(`which ${name}`, { stdio: "pipe" }).toString().trim();
				if (p) return { path: p, profileDir: `${process.env.HOME}/.config/google-chrome/` };
			} catch {}
		}
	}
	console.error("✗ No Chrome or Chromium found. Install with: sudo apt install chromium-browser");
	process.exit(1);
}

const browser_info = findBrowser();
const SCRAPING_DIR = `${process.env.HOME}/.cache/browser-tools`;

// Check if already running on :9222
try {
	const browser = await puppeteer.connect({
		browserURL: "http://localhost:9222",
		defaultViewport: null,
	});
	await browser.disconnect();
	console.log("✓ Chrome already running on :9222");
	process.exit(0);
} catch {}

// Setup profile directory
execSync(`mkdir -p "${SCRAPING_DIR}"`, { stdio: "ignore" });

// Remove SingletonLock to allow new instance
try {
	execSync(`rm -f "${SCRAPING_DIR}/SingletonLock" "${SCRAPING_DIR}/SingletonSocket" "${SCRAPING_DIR}/SingletonCookie"`, { stdio: "ignore" });
} catch {}

if (useProfile) {
	console.log("Syncing profile...");
	execSync(
		`rsync -a --delete \
			--exclude='SingletonLock' \
			--exclude='SingletonSocket' \
			--exclude='SingletonCookie' \
			--exclude='*/Sessions/*' \
			--exclude='*/Current Session' \
			--exclude='*/Current Tabs' \
			--exclude='*/Last Session' \
			--exclude='*/Last Tabs' \
			"${browser_info.profileDir}" "${SCRAPING_DIR}/"`,
		{ stdio: "pipe" },
	);
}

// Start Chrome/Chromium with flags to force new instance
const chromeArgs = [
	"--remote-debugging-port=9222",
	`--user-data-dir=${SCRAPING_DIR}`,
	"--no-first-run",
	"--no-default-browser-check",
];
// Headless on Linux if no DISPLAY
if (platform() !== "darwin" && !process.env.DISPLAY && !process.env.WAYLAND_DISPLAY) {
	chromeArgs.unshift("--headless=new", "--no-sandbox");
}

spawn(browser_info.path, chromeArgs, { detached: true, stdio: "ignore" }).unref();

// Wait for Chrome to be ready
let connected = false;
for (let i = 0; i < 30; i++) {
	try {
		const browser = await puppeteer.connect({
			browserURL: "http://localhost:9222",
			defaultViewport: null,
		});
		await browser.disconnect();
		connected = true;
		break;
	} catch {
		await new Promise((r) => setTimeout(r, 500));
	}
}

if (!connected) {
	console.error("✗ Failed to connect to Chrome");
	process.exit(1);
}

console.log(`✓ Chrome started on :9222${useProfile ? " with your profile" : ""}`);
