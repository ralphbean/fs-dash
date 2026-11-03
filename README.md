# fs-dash

A Go/Bubble Tea terminal dashboard for finding the next useful action on Fullsend pull requests. The first change provides a fast PR list, fetch-state indicators, local command bindings, and an in-app log console. Later changes are listed in [the OpenSpec roadmap](openspec/ROADMAP.md).

## Run

Requires Go 1.26 or newer and an authenticated [GitHub CLI](https://cli.github.com/).

    go run -buildvcs=false ./cmd/fs-dash --repo fullsend-ai/fullsend

Without `--repo`, fs-dash uses `repositories` from `~/.config/fs-dash/config.toml`, then `GH_REPO`, then the current directory's GitHub `origin` remote. If none is available, it shows a setup message.

Copy [config.example.toml](config.example.toml) to `~/.config/fs-dash/config.toml` to configure repositories and hotkeys. The default `o` binding opens the selected PR URL with `xdg-open` on Linux or `open` on macOS. Bindings are executable paths and argument arrays; placeholders such as `{url}` are expanded without a shell. A custom launcher can use `{url}` in a prompt and set `dir` to its working directory.

Keys: `j/k` or arrows move, Enter opens the preliminary details view, `r` refreshes, `L` opens logs, `?` shows help, and `q` quits. Logs stay in a bounded in-memory buffer and are not written to disk.
