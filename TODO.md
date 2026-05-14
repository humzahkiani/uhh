# TODO

Internal tasks: refactors, tooling, code-quality work. User-facing features live in `USER_FLOWS.md`.

## Backlog

1. Write tests
2. Consider a `UHH_CONFIG` env var to override the config path for testing and oddball setups.3. Create lint CI

## Done

1. Add `.golangci.yml` with `revive` + full `staticcheck` checks enabled.
2. Move config from CWD `./commands.yaml` to `~/.config/uhh/commands.yaml` (XDG-style, with `$XDG_CONFIG_HOME` override).
3. Refactor `main` into the `run() error` pattern — main is now a thin error-to-exit-code translator.
4. Add `commands.example.yaml` for first-time users.
