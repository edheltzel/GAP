#!/usr/bin/env node

import puppeteer from "puppeteer-core";

const args = process.argv.slice(2);

// Named presets
const presets = {
	"iphone-se": { width: 375, height: 667, deviceScaleFactor: 2, mobile: true },
	iphone: { width: 390, height: 844, deviceScaleFactor: 3, mobile: true },
	"iphone-pro-max": { width: 430, height: 932, deviceScaleFactor: 3, mobile: true },
	ipad: { width: 820, height: 1180, deviceScaleFactor: 2, mobile: true },
	"ipad-pro": { width: 1024, height: 1366, deviceScaleFactor: 2, mobile: true },
	android: { width: 412, height: 915, deviceScaleFactor: 2.625, mobile: true },
	tablet: { width: 768, height: 1024, deviceScaleFactor: 2, mobile: true },
	laptop: { width: 1366, height: 768, deviceScaleFactor: 1, mobile: false },
	desktop: { width: 1920, height: 1080, deviceScaleFactor: 1, mobile: false },
};

function printUsage() {
	console.log("Usage: browser-resize.js <width>x<height> [--dpr=N] [--mobile]");
	console.log("       browser-resize.js <preset>");
	console.log("       browser-resize.js reset");
	console.log("");
	console.log("Examples:");
	console.log("  browser-resize.js 375x812              # Custom size");
	console.log("  browser-resize.js 375x812 --dpr=2      # With device pixel ratio");
	console.log("  browser-resize.js 375x812 --mobile     # With mobile emulation");
	console.log("  browser-resize.js iphone                # Named preset");
	console.log("  browser-resize.js reset                 # Clear overrides");
	console.log("");
	console.log("Presets:");
	for (const [name, p] of Object.entries(presets)) {
		console.log(`  ${name.padEnd(18)} ${p.width}x${p.height} @${p.deviceScaleFactor}x${p.mobile ? " (mobile)" : ""}`);
	}
	process.exit(1);
}

if (args.length === 0) printUsage();

const b = await Promise.race([
	puppeteer.connect({ browserURL: "http://localhost:9222", defaultViewport: null }),
	new Promise((_, reject) => setTimeout(() => reject(new Error("timeout")), 5000)),
]).catch((e) => {
	console.error("✗ Could not connect to browser:", e.message);
	console.error("  Run: browser-start.js");
	process.exit(1);
});

const p = (await b.pages()).at(-1);
if (!p) {
	console.error("✗ No active tab found");
	process.exit(1);
}

const client = await p.createCDPSession();

// Reset
if (args[0] === "reset") {
	await client.send("Emulation.clearDeviceMetricsOverride");
	console.log("✓ Viewport reset to default");
	await b.disconnect();
	process.exit(0);
}

// Preset
if (presets[args[0]]) {
	const preset = presets[args[0]];
	await client.send("Emulation.setDeviceMetricsOverride", preset);
	console.log(`✓ Viewport: ${preset.width}x${preset.height} @${preset.deviceScaleFactor}x${preset.mobile ? " (mobile)" : ""} [${args[0]}]`);
	await b.disconnect();
	process.exit(0);
}

// Custom WxH
const sizeMatch = args[0].match(/^(\d+)x(\d+)$/);
if (!sizeMatch) {
	console.error(`✗ Unknown preset or invalid size: ${args[0]}`);
	console.error('  Use WxH format (e.g. "375x812") or a preset name');
	printUsage();
}

const width = parseInt(sizeMatch[1]);
const height = parseInt(sizeMatch[2]);
let deviceScaleFactor = 1;
let mobile = false;

for (const arg of args.slice(1)) {
	if (arg.startsWith("--dpr=")) {
		deviceScaleFactor = parseFloat(arg.slice(6));
	} else if (arg === "--mobile") {
		mobile = true;
	}
}

await client.send("Emulation.setDeviceMetricsOverride", { width, height, deviceScaleFactor, mobile });
console.log(`✓ Viewport: ${width}x${height} @${deviceScaleFactor}x${mobile ? " (mobile)" : ""}`);
await b.disconnect();
