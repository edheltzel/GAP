---
name: BunGettingStarted
description: Provides comprehensive guidance for getting started with Bun 1.3+ (latest version), including installation, project setup with HRM (Hot Module Replacement), database configuration with Bun.SQL (MySQL, PostgreSQL, SQLite), and best practices for building full-stack JavaScript applications.
tags: [bun, javascript, full-stack, databases, development]
---

# Bun 1.3 Getting Started Guide

## Overview

Bun 1.3+(latest version) is a batteries-included full-stack JavaScript runtime that combines a fast JavaScript runtime with built-in tooling for frontend and backend development. This guide walks through everything needed to get a new project running smoothly.

## Installation

### macOS (Recommended for your setup)

```bash
# Using Homebrew (simplest)
brew tap oven-sh/bun
brew install bun

# Or using the install script
curl -fsSL https://bun.sh/install | bash

# Verify installation
bun --version
```

### Other Installation Methods

```bash
# npm
npm install -g bun

# Docker
docker pull oven/bun
docker run --rm --init --ulimit memlock=-1:-1 oven/bun

# Windows (PowerShell)
powershell -c "irm bun.sh/install.ps1 | iex"
```

## Project Initialization

### Create a New Project

```bash
# Blank project
bun init

# React project (with HMR built-in)
bun init --react

# React with Tailwind
bun init --react=tailwind

# React with shadcn/ui
bun init --react=shadcn
```

### Project Structure

A typical Bun full-stack project looks like:

```
my-app/
├── src/
│   ├── index.html          # Frontend entry point
│   ├── App.tsx             # React component
│   ├── styles.css          # Frontend styles
│   └── server.ts           # Backend routes
├── bun.lock                # Dependency lock file
├── bunfig.toml             # Bun configuration
├── tsconfig.json           # TypeScript configuration
└── package.json            # Dependencies & scripts
```

## Hot Module Replacement (HMR)

### What is HMR?

Hot Module Replacement lets you update your code and see changes instantly without page reloads or losing app state. Bun 1.3+ (latest version) includes built-in HMR support for both frontend and backend development.

### Setting Up HMR in Your Server

```typescript
// server.ts
import { serve } from "bun";
import App from "./App.html";

serve({
  port: 3000,
  development: {
    // Enable Hot Module Reloading
    hmr: true,

    // Echo browser console logs to terminal
    console: true,
  },
  routes: {
    "/": App,
    "/api/*": handleApi,
  },
});

async function handleApi(request: Request) {
  return Response.json({ message: "Hello from API" });
}
```

### HMR Features

- **React Fast Refresh**: Update React components without losing state
- **Browser Console Logs**: See `console.log()` output from browser in your terminal
- **Automatic Reloading**: Changes are reflected instantly
- **Zero Configuration**: Works out of the box with `hmr: true`

### Client-Side HMR API

For custom HMR behavior:

```typescript
if (import.meta.hot) {
  import.meta.hot.accept(() => {
    console.log("Module updated!");
  });

  import.meta.hot.dispose(() => {
    console.log("Cleaning up old module");
  });
}
```

## Database Setup with Bun.SQL

### Overview

Bun.SQL provides a unified API for MySQL, PostgreSQL, and SQLite with zero dependencies. It's incredibly fast and type-safe.

### 1. PostgreSQL Setup

#### Installation & Connection

```bash
# macOS - install PostgreSQL
brew install postgresql

# Start PostgreSQL service
brew services start postgresql

# Create a database
createdb myapp_db

# Or use Docker
docker run --name postgres -e POSTGRES_PASSWORD=password -d postgres
```

#### Usage in Bun

```typescript
import { sql } from "bun";

// Connection from DATABASE_URL environment variable (recommended)
// export DATABASE_URL="postgres://user:password@localhost/mydb"
const db = sql;

// Or create explicitly
import { SQL } from "bun";
const db = new SQL("postgres://localhost/mydb");

// Simple query
const users = await sql`SELECT * FROM users LIMIT 10`;

// Parameterized query (safe from SQL injection)
const age = 65;
const seniors = await sql`
  SELECT name, age FROM users
  WHERE age >= ${age}
`;

// Insert with returning
const [user] = await sql`
  INSERT INTO users (name, email)
  VALUES (${"Alice"}, ${"alice@example.com"})
  RETURNING *
`;

// Update
await sql`
  UPDATE users
  SET name = ${"Bob"}
  WHERE id = ${userId}
`;

// Working with arrays (PostgreSQL specific)
await sql`
  INSERT INTO users (name, roles)
  VALUES (${"Alice"}, ${sql.array(["admin", "user"], "TEXT")})
`;
```

#### Advanced PostgreSQL Features

```typescript
// Multi-statement queries (for migrations)
await sql`
  CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    email TEXT UNIQUE NOT NULL
  );
  CREATE INDEX idx_users_email ON users(email);
  INSERT INTO users (name, email) VALUES ('Admin', 'admin@example.com');
`.simple();

// Unix domain socket (faster on same machine)
const db = new SQL({
  path: "/tmp/.s.PGSQL.5432",
  user: "postgres",
  password: "postgres",
  database: "mydb",
});

// Disable prepared statements (for PGBouncer)
const db = new SQL({
  prepare: false,
});

// Dynamic column operations
const user = { name: "Alice", email: "alice@example.com", age: 30 };
await sql`INSERT INTO users ${sql(user, "name", "email")}`;

const updates = { name: "Alice Smith", email: "alice.smith@example.com" };
await sql`UPDATE users SET ${sql(updates)} WHERE id = ${userId}`;
```

### 2. MySQL Setup

#### Installation & Connection

```bash
# macOS
brew install mysql

# Start MySQL
brew services start mysql

# Create database
mysql -u root -e "CREATE DATABASE myapp_db;"

# Or use Docker
docker run --name mysql -e MYSQL_ROOT_PASSWORD=password -d mysql
```

#### Usage in Bun

```typescript
import { SQL } from "bun";

// MySQL connection
const db = new SQL("mysql://root:password@localhost/myapp_db");

// Queries work the same as PostgreSQL
const users = await db`SELECT * FROM users LIMIT 10`;

const [user] = await db`
  INSERT INTO users (name, email)
  VALUES (${"Bob"}, ${"bob@example.com"})
`;
```

### 3. SQLite Setup

#### Installation & Usage

SQLite is the easiest option—just a file on disk, no server needed.

```typescript
import { SQL } from "bun";

// Create/connect to SQLite database (creates file if not exists)
const db = new SQL("sqlite://data.db");

// All standard operations work
const users = await db`SELECT * FROM users`;

// SQLite-specific: Deserialize with options
import { Database } from "bun:sqlite";
const serialized = db.serialize();
const deserialized = Database.deserialize(serialized, {
  readonly: true,
  strict: true,
  safeIntegers: true,
});

// Column type introspection
const stmt = db.query("SELECT * FROM users");
console.log(stmt.declaredTypes); // ["INTEGER", "TEXT", "INTEGER"]
console.log(stmt.columnTypes); // ["integer", "text", "integer"]
```

### Choosing Your Database

| Database       | Best For                                                         | Setup Complexity |
| -------------- | ---------------------------------------------------------------- | ---------------- |
| **PostgreSQL** | Production servers, advanced features (arrays, JSON), multi-user | Medium           |
| **MySQL**      | Production servers, web apps, reliability                        | Medium           |
| **SQLite**     | Development, single-file apps, mobile, embedded                  | Easy             |

### Database Connection Patterns

#### Environment-Based Configuration

```typescript
// bunfig.toml
[install]
# Automatically load from DATABASE_URL
# export DATABASE_URL="postgres://localhost/mydb"

// server.ts
import { sql } from "bun";

// Automatically uses DATABASE_URL from environment
const result = await sql`SELECT version()`;
```

#### Connection Pooling

```typescript
import { SQL } from "bun";

// Configure connection pooling
const db = new SQL("postgres://localhost/mydb", {
  max: 20, // Maximum connections in pool
});
```

## Building Full-Stack Apps

### Basic Full-Stack Example

```typescript
// server.ts
import { serve } from "bun";
import { sql } from "bun";
import App from "./App.html";

serve({
  port: 3000,
  development: {
    hmr: true,
    console: true,
  },
  routes: {
    // Serve React app on root and /dashboard
    "/*": App,

    // API routes
    "/api/users": {
      GET: async () => {
        const users = await sql`SELECT * FROM users LIMIT 10`;
        return Response.json(users);
      },
      POST: async (req) => {
        const { name, email } = await req.json();
        const [user] = await sql`
          INSERT INTO users ${sql({ name, email })}
          RETURNING *
        `;
        return Response.json(user);
      },
    },

    // Dynamic routes
    "/api/users/:id": async (req) => {
      const { id } = req.params;
      const [user] = await sql`
        SELECT * FROM users WHERE id = ${id} LIMIT 1
      `;

      if (!user) {
        return new Response("User not found", { status: 404 });
      }

      return Response.json(user);
    },

    // Health check
    "/healthcheck.json": Response.json({ status: "ok" }),
  },
});
```

### Cookies

```typescript
import { serve, randomUUIDv7 } from "bun";

serve({
  routes: {
    "/api/login": (request) => {
      // Set cookie automatically
      request.cookies.set("sessionId", randomUUIDv7(), {
        httpOnly: true,
        sameSite: "strict",
        maxAge: 60 * 60 * 24 * 7, // 7 days
      });
      return new Response("Logged in");
    },

    "/api/logout": (request) => {
      // Delete cookie
      request.cookies.delete("sessionId");
      return new Response("Logged out");
    },

    "/api/profile": (request) => {
      // Read cookie
      const sessionId = request.cookies.get("sessionId");
      if (!sessionId) {
        return new Response("Not authenticated", { status: 401 });
      }
      return Response.json({ session: sessionId });
    },
  },
});
```

### WebSockets

```typescript
import { serve } from "bun";

serve({
  routes: {
    "/ws": {
      fetch(req) {
        if (req.headers.get("upgrade") === "websocket") {
          return new Response(null, {
            status: 101,
            webSocket: {
              open(ws) {
                console.log("Client connected");
                ws.send(JSON.stringify({ message: "Welcome!" }));
              },
              message(ws, message) {
                console.log("Received:", message);
                ws.send(JSON.stringify({ echo: message }));
              },
              close(ws) {
                console.log("Client disconnected");
              },
            },
          });
        }
        return new Response("Not a WebSocket request", { status: 400 });
      },
    },
  },
});
```

## Package Management with Bun

### Installing Dependencies

```bash
# Install all dependencies from package.json
bun install

# Add a package
bun add express
bun add -d @types/node

# Install from specific version
bun add react@18.3.1

# Install optional or peer dependencies
bun add --optional webpack
```

### Workspace Monorepos

```json
{
  "name": "monorepo",
  "workspaces": ["packages/*"],
  "catalogs": {
    "react": "^18.0.0",
    "typescript": "^5.0.0",
    "tailwindcss": "^3.0.0"
  }
}
```

In workspace packages, reference catalog versions:

```json
{
  "name": "@company/frontend",
  "dependencies": {
    "react": "catalog:react",
    "tailwindcss": "catalog:tailwindcss"
  }
}
```

### Useful Commands

```bash
# Check what depends on a package
bun why react

# Interactive dependency updates
bun update --interactive

# Recursive updates in monorepo
bun update --interactive --filter @company/frontend

# Audit for vulnerabilities
bun audit

# Configure security scanner
# Add to bunfig.toml:
# [install.security]
# scanner = "@socketsecurity/bun-security-scanner"
```

## TypeScript Configuration

### Default Setup

```json
{
  "compilerOptions": {
    "module": "Preserve",
    "lib": ["ESNext"],
    "target": "ESNext",
    "moduleResolution": "bundler",
    "allowJs": true,
    "jsx": "react-jsx"
  }
}
```

### Environment Types

Bun automatically detects whether to use Node.js or DOM types. Override if needed:

```json
{
  "compilerOptions": {
    "types": ["bun-types"]
  }
}
```

## Building for Production

### Create a Standalone Executable

```bash
# Bundle and compile to executable
bun build --compile ./server.ts --outfile myapp

# With specific target
bun build --compile --target=linux-x64 ./server.ts --outfile myapp-linux

# Run the executable
./myapp
```

### Environment Variables

```bash
# Create .env file
DATABASE_URL="postgres://user:pass@localhost/mydb"
NODE_ENV="production"
PORT="8080"

# Load in your app (automatic with Bun)
const dbUrl = process.env.DATABASE_URL;
```

## Testing

### Basic Test Setup

```bash
# Bun comes with built-in test runner
bun test
```

```typescript
// math.test.ts
import { test, expect } from "bun:test";

test("addition", () => {
  expect(1 + 1).toBe(2);
});

test("async operations", async () => {
  const result = await Promise.resolve(42);
  expect(result).toBe(42);
});

// Concurrent tests (for I/O bound operations)
test.concurrent("fetch users", async () => {
  const res = await fetch("/api/users");
  expect(res.status).toBe(200);
});
```

## Performance Tips

1. **Use Prepared Statements**: Bun.SQL uses them by default (safest)
2. **Connection Pooling**: Configure `max` option for database connections
3. **Lazy Load Routes**: Use dynamic imports for rarely-used features
4. **Cache Static Assets**: Let Bun serve them directly
5. **Monitor Memory**: Use `--console-depth` for debugging
6. **Database Preconnection**: Use `--sql-preconnect` flag to reduce first-query latency

```bash
export DATABASE_URL="postgres://localhost/mydb"
bun --sql-preconnect ./server.ts
```

## Common Patterns

### API Response Wrapper

```typescript
function apiResponse<T>(data: T, status = 200) {
  return Response.json({ data, status }, { status });
}

function apiError(message: string, status = 400) {
  return Response.json({ error: message, status }, { status });
}
```

### Database Migrations

```typescript
// migrations.ts
import { sql } from "bun";

export async function migrate() {
  // Create tables
  await sql`
    CREATE TABLE IF NOT EXISTS users (
      id SERIAL PRIMARY KEY,
      name TEXT NOT NULL,
      email TEXT UNIQUE NOT NULL,
      created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
    );

    CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
  `.simple();

  console.log("✓ Migrations complete");
}

// In server.ts
await migrate();
```

### Error Handling

```typescript
try {
  const [user] = await sql`
    INSERT INTO users (email) VALUES (${email})
  `;
  return Response.json(user);
} catch (error) {
  if (error instanceof Error) {
    if (error.message.includes("unique")) {
      return apiError("Email already exists", 409);
    }
  }
  console.error(error);
  return apiError("Internal server error", 500);
}
```

## Troubleshooting

### Database Connection Issues

```bash
# Test connection manually
echo "SELECT 1" | psql postgres://user:pass@localhost/dbname

# Check environment variables
echo $DATABASE_URL

# Increase verbosity
export BUN_CONFIG_VERBOSE_FETCH=curl
```

### Hot Reload Not Working

- Ensure `hmr: true` is set in `Bun.serve()`
- Check that `development` object is properly configured
- Verify file changes are being saved to disk
- Clear browser cache or open in private/incognito mode

### Module Not Found Errors

```bash
# Install missing dependencies
bun install

# Check if dependency is in package.json
bun why package-name

# Reinstall with clean node_modules
rm -rf node_modules bun.lock
bun install
```

## Resources

- **Official Docs**: https://bun.sh/docs
- **GitHub**: https://github.com/oven-sh/bun
- **Discord Community**: https://bun.sh/discord
- **Bun Blog**: https://bun.sh/blog

## Next Steps

1. Create your first project: `bun init --react`
2. Set up your database connection
3. Build API routes with `Bun.serve()`
4. Deploy using `bun build --compile`
5. Monitor performance and optimize as needed

---

**Version History**

- v1.0 - Initial guide for Bun 1.3 release
- Includes HMR setup, Bun.SQL for all three databases, full-stack examples
