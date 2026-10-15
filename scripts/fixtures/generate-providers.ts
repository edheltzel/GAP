import { mock } from "bun:test";

const image = "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO+a9m8AAAAASUVORK5CYII=";

mock.module("openai", () => ({
  default: class {
    images = { generate: async ({ n }: { n: number }) => ({ data: Array.from({ length: n }, () => ({ b64_json: image })) }) };
  },
}));

mock.module("@google/genai", () => ({
  GoogleGenAI: class {
    models = { generateContent: async () => ({ candidates: [{ content: { parts: [{ inlineData: { data: image } }] } }] }) };
  },
}));
