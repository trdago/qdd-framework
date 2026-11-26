# QDD Framework: rules for AI assistants

This repository is governed by QDD itself. Full rules: `AGENTS.md`. The essentials:

1. **Context first.** Before changing code, run `qdd modules context <paths>` (or the MCP tool `qdd_project_context`) and
   respect the module record: its goal, decisions and invariants.
2. **Tests first.** Write the failing test, see it fail, then implement.
3. **Zero `else`.** Guard clauses on top, straight main path. Fewer people read code line by line, so what is read must be
   understood at a glance.
4. **Every bug is a finding** in `.qdd/project/findings/` with its regression test. A `fix:` commit names its `FND-NNN`.
5. **Never commit or deploy unless `qdd audit` passes.** Build with `make build`; the binary is `./qdd` in the root.
6. **Publishing is a person's decision** (tag, `npm publish`, PR merge): queue it with `qdd consola pedir`.
7. **Temporary files go in `./tmp`.** No real names of organizations, clients or projects in the docs: the repository is public.
