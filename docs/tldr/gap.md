# gap

> Publish one set of agent rules, commands, and skills to every coding agent.
> More information: <https://gap.edheltzel.codes/docs/>.

- Open the interactive terminal UI:

`gap`

- Preview and publish the policy, commands, and component skills:

`gap sync`

- Sync only some groups, skipping the group picker:

`gap sync --only {{policy,component_id}} --yes`

- Show each component's pin, store, and skill drift:

`gap status`

- Add skills from a GitHub repository for every project:

`gap add -g {{owner}}/{{repo}}`

- Update an added utility to the newest commit:

`gap update {{utility_id}}`

- Reinstall drifted skills from the lock:

`gap restore`

- Explain the terms gap uses:

`gap help concepts`
