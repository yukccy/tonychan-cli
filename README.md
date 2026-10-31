# tonychan

My site, [yukccy.com](https://yukccy.com), in a terminal: who I am, my work, and everything I've
written, one command per page.

```
$ tonychan
████████╗ ██████╗ ███╗   ██╗██╗   ██╗   ██████╗██╗  ██╗ █████╗ ███╗   ██╗
╚══██╔══╝██╔═══██╗████╗  ██║╚██╗ ██╔╝  ██╔════╝██║  ██║██╔══██╗████╗  ██║
   ██║   ██║   ██║██╔██╗ ██║ ╚████╔╝   ██║     ███████║███████║██╔██╗ ██║
   ██║   ██║   ██║██║╚██╗██║  ╚██╔╝    ██║     ██╔══██║██╔══██║██║╚██╗██║
   ██║   ╚██████╔╝██║ ╚████║   ██║     ╚██████╗██║  ██║██║  ██║██║ ╚████║██╗
   ╚═╝    ╚═════╝ ╚═╝  ╚═══╝   ╚═╝      ╚═════╝╚═╝  ╚═╝╚═╝  ╚═╝╚═╝  ╚═══╝╚═╝

NOW          Member of Technical Staff, OpusClip · since Mar 2026 · Vancouver
LATEST POST  Private Kubernetes Access with Tailscale · 2025-10-15

LOG
* │  2026-03     WORK     Joined OpusClip as Member of Technical Staff
│ ●  2025-10-15  WRITING  Private Kubernetes Access with Tailscale
…
```

## Install

```sh
brew install yukccy/tap/tonychan
```

or, with Go 1.24 or later:

```sh
go install github.com/yukccy/tonychan-cli/cmd/tonychan@latest
```

Binaries for macOS, Linux and Windows are on the
[releases page](https://github.com/yukccy/tonychan-cli/releases).

## Use

| Command | Shows | On the site |
|---|---|---|
| `tonychan` | Home: now, latest post, and the log of posts and work | [yukccy.com](https://yukccy.com) |
| `tonychan --work` / `--writing` | The log, filtered | |
| `tonychan work [--all]` | Roles, certifications (`--all` adds expired ones), education | [/work](https://yukccy.com/work/) |
| `tonychan writing [--all] [--tag <topic>]` | Posts, newest first | [/writing](https://yukccy.com/writing/) |
| `tonychan writing <slug or URL>` | One post, rendered and paged | `/writing/<slug>` |
| `tonychan open [slug or URL]` | The site, or a post, in your browser | |
| `tonychan contact` | Email and profiles | |

Post links are clickable in terminals that support hyperlinks. Colour follows
[`NO_COLOR`](https://no-color.org), and posts page through `$PAGER` (default `less -RFX`;
`--no-pager` prints instead).

## How it works

There is no content in this repo. The site's Hugo build publishes
[`/index.json`](https://yukccy.com/index.json) (work, log and the post list) and every post as
Markdown (`/writing/<slug>/index.md`); the CLI reads those at run time, so a new post shows up
without a new release. The last copy is cached, so it still works offline.

`TONYCHAN_SITE` points it at another copy of the site, such as a local `hugo server`:

```sh
TONYCHAN_SITE=http://localhost:1313 go run ./cmd/tonychan
```

## Release

Push a `v*` tag. GoReleaser builds the binaries, publishes the GitHub release and updates the
Homebrew cask in [yukccy/homebrew-tap](https://github.com/yukccy/homebrew-tap), using the
`HOMEBREW_TAP_GITHUB_TOKEN` secret.
