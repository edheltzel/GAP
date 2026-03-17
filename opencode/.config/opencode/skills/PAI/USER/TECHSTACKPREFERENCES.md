# Technology Stack Preferences

## Runtime & Package Manager

- **Runtime:** Bun (preferred)
- **Package Manager:** Bun

## Languages

- **Primary:** TypeScript, PHP
- **Emerging:** Go (building momentum, future direction)
- **Supporting:** JavaScript, HTML, CSS

## Cloud & Deployment

- **Platform:** Cloudflare (primary — Workers, Pages, KV, D1)
- **Services:** Cloudflare Workers, Cloudflare Pages, Cloudflare KV, Cloudflare D1
- **Deployment Method:** Wrangler CLI

## Frontend

- **Framework:** No fixed preference — varies by project (React, Vue, Svelte, etc.)
- **Styling:** Tailwind CSS (preferred). Historical use of SCSS, PicoCSS, Bootstrap.
- **Approach:** Project-appropriate selection over dogmatic framework loyalty

## Backend

- **PHP:** Laravel ecosystem (when PHP is the right tool)
- **TypeScript:** Bun-native, Cloudflare Workers
- **Go:** Growing adoption for performance-critical services

## Databases

- **Relational:** MySQL / MariaDB, PostgreSQL, SQLite
- **Edge/Serverless:** Cloudflare D1, Cloudflare KV
- **BaaS:** Supabase, Firebase
- **Preference:** Match database to deployment context (edge = D1/KV, traditional = MySQL/Postgres, rapid prototype = Supabase)

## Testing

- **TypeScript/JS:** Vitest (preferred), Jest
- **PHP:** PHPUnit, PHP CS Fixer / Laravel Pint
- **E2E:** Open to exploration (Playwright, Cypress)
- **General approach:** Open to adopting best-fit testing tools per project

## Conventions

- **Formatting (TS/JS):** Biome (preferred — fast, all-in-one)
- **Formatting (PHP):** PHP CS Fixer / Laravel Pint
- **File Naming:** Project-contextual (kebab-case for web, PascalCase for components)

## Design & Creative Tools

- **Design:** Figma, Adobe Creative Suite
- **3D:** 3D modeling tools for printing (functional parts, keyboard enclosures)
- **Prototyping:** Component-based UI design, wireframing
