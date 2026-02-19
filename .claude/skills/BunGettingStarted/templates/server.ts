import { serve } from "bun";
import { sql } from "bun";
import App from "./App.html";

// Simple API endpoint handler
async function handleApi(req: Request, pathname: string) {
  const url = new URL(req.url);

  // GET /api/health
  if (pathname === "/api/health" && req.method === "GET") {
    return Response.json({ status: "ok", timestamp: new Date() });
  }

  // GET /api/users
  if (pathname === "/api/users" && req.method === "GET") {
    try {
      const users = await sql`SELECT * FROM users LIMIT 10`;
      return Response.json(users);
    } catch (error) {
      console.error("Database error:", error);
      return Response.json({ error: "Failed to fetch users" }, { status: 500 });
    }
  }

  // POST /api/users
  if (pathname === "/api/users" && req.method === "POST") {
    try {
      const { name, email } = await req.json();

      if (!name || !email) {
        return Response.json({ error: "Name and email required" }, { status: 400 });
      }

      const [user] = await sql`
        INSERT INTO users ${sql({ name, email })}
        RETURNING *
      `;

      return Response.json(user, { status: 201 });
    } catch (error) {
      console.error("Error creating user:", error);
      return Response.json({ error: "Failed to create user" }, { status: 500 });
    }
  }

  // GET /api/users/:id
  if (pathname.match(/^\/api\/users\/\d+$/) && req.method === "GET") {
    const id = pathname.split("/").pop();
    try {
      const [user] = await sql`SELECT * FROM users WHERE id = ${id} LIMIT 1`;

      if (!user) {
        return Response.json({ error: "User not found" }, { status: 404 });
      }

      return Response.json(user);
    } catch (error) {
      console.error("Error fetching user:", error);
      return Response.json({ error: "Failed to fetch user" }, { status: 500 });
    }
  }

  return Response.json({ error: "Not found" }, { status: 404 });
}

serve({
  port: parseInt(process.env.PORT || "3000"),
  development: {
    // Enable Hot Module Replacement
    hmr: true,

    // Echo browser console.log() to terminal
    console: true,
  },
  async fetch(request) {
    const url = new URL(request.url);
    const pathname = url.pathname;

    // API routes
    if (pathname.startsWith("/api/")) {
      return handleApi(request, pathname);
    }

    // Health check
    if (pathname === "/healthcheck.json") {
      return Response.json({ status: "ok" });
    }

    // Serve HTML app for all other routes (SPA routing)
    return new Response(Bun.file("./public/index.html"));
  },
});

console.log("🔥 Server running at http://localhost:3000");
console.log("🔥 HMR enabled - changes will reload automatically");
