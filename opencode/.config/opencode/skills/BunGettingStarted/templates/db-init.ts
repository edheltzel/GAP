import { sql } from "bun";

/**
 * Database initialization script for Bun 1.3
 * Supports: PostgreSQL, MySQL, SQLite
 * 
 * Usage:
 *   bun run db/init.ts
 */

export async function initializeDatabase() {
  console.log("🗄️  Initializing database...\n");

  try {
    // Create users table
    await sql`
      CREATE TABLE IF NOT EXISTS users (
        id SERIAL PRIMARY KEY,
        name VARCHAR(255) NOT NULL,
        email VARCHAR(255) UNIQUE NOT NULL,
        created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
        updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
      )
    `.simple();

    console.log("✓ Created users table");

    // Create posts table
    await sql`
      CREATE TABLE IF NOT EXISTS posts (
        id SERIAL PRIMARY KEY,
        user_id INTEGER NOT NULL,
        title VARCHAR(255) NOT NULL,
        content TEXT NOT NULL,
        created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
        updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
        FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
      )
    `.simple();

    console.log("✓ Created posts table");

    // Create indexes
    await sql`
      CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
      CREATE INDEX IF NOT EXISTS idx_posts_user_id ON posts(user_id);
    `.simple();

    console.log("✓ Created indexes");

    // Seed sample data
    const userCount = await sql`SELECT COUNT(*) as count FROM users`;
    if (userCount[0].count === 0) {
      await sql`
        INSERT INTO users (name, email) VALUES
        (${"Alice"}, ${"alice@example.com"}),
        (${"Bob"}, ${"bob@example.com"}),
        (${"Charlie"}, ${"charlie@example.com"})
      `;

      console.log("✓ Seeded sample users");

      await sql`
        INSERT INTO posts (user_id, title, content) VALUES
        (1, ${"Hello World"}, ${"This is my first post"}),
        (2, ${"Bun is Fast"}, ${"I'm impressed with Bun's performance"}),
        (1, ${"TypeScript Love"}, ${"TypeScript makes development easier"})
      `;

      console.log("✓ Seeded sample posts");
    } else {
      console.log("✓ Database already contains data, skipping seed");
    }

    console.log("\n✨ Database initialization complete!");
  } catch (error) {
    console.error("❌ Database initialization failed:", error);
    process.exit(1);
  }
}

// Run if this is the main module
if (import.meta.main) {
  await initializeDatabase();
}

export default initializeDatabase;
