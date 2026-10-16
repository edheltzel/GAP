# CreateCLI

Quickstart:

```bash
npx skills add edheltzel/Do-Skills --skill=do-create-cli
```

```bash
npx skills update do-create-cli
```

[Source](https://github.com/edheltzel/Do-Skills/tree/master/skills/project/tooling/do-create-cli)

## What it does

CreateCLI generates a production-ready TypeScript command-line tool, including
its implementation, Bun package setup, strict TypeScript configuration,
documentation, JSON output, and exit-code behavior. It also routes work to
creating a CLI, adding a command, or upgrading the CLI's complexity tier.

Its defining constraint is to start with the simplest tier that fits: manual
argument parsing is the default, Commander.js is for grouped or nested command
surfaces, and oclif is reference-only for rare enterprise-scale cases.

## When to reach for it

Type `/do-create-cli`, or the agent reaches for it automatically when a request is
to create a TypeScript CLI, wrap an API, add a command, or replace a complex
shell script with a command-line tool.

Reach for it when the deliverable is a usable command-line interface rather than
an application feature. For the engineering standards that guide TypeScript and
Effect changes more broadly, use
[coding-standards](../typescript/ts-standards.md).

## A CLI-first delivery

The generated interface is designed to be deterministic and composable: the
same input produces the same JSON output, so people and other tools can reliably
pipe it onward. Each generated CLI includes help and usage material alongside
configuration, typed source, actionable errors, and compliant success or error
exit codes.

Tier selection follows the command surface, not anticipated prestige. Simple
API clients, data transformers, and automation tools generally stay with a
small zero-dependency parser; complex subcommands and nested options earn the
Commander.js tier.

## Where it fits

CreateCLI is a project-creation skill for command-line tools. It produces a
complete CLI rather than a general TypeScript coding style; apply
[write-typescript](../typescript/write-typescript.md) for day-to-day TypeScript authoring
and [lean-ts-patterns](./lean-ts-patterns.md) when lightweight,
zero-dependency TypeScript patterns are the central concern.
