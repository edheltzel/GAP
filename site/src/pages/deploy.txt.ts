export const prerender = true;

export function GET() {
  return new Response(`${process.env.CF_PAGES_COMMIT_SHA ?? "local"}\n`, {
    headers: { "Content-Type": "text/plain; charset=utf-8" },
  });
}
