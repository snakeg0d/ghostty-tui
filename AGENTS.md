<!-- workbench:begin (managed by ~/dev/tools/workbench/sync-rules; edit the sources, not this block) -->
Venture: Personal (personal in snakeg0d/dev projects.yaml). Project: Ghostty TUI (snakeg0d/ghostty-tui, paused). Hub: snakeg0d/personal-workspace. Ritual: no.

## Workbench rules (copied from snakeg0d/dev AGENTS.md)

## How Daniel works
- He says what he wants in plain words, often from his phone. Use the tools that already exist, then reply briefly: what changed and where, plus anything that failed or needs his decision.
- He approves in words. Do reversible work within his request without asking, except the actions under "Needs Daniel's yes".
- Talk like a colleague, in plain words: no nicknames, no fixed report formats, no filler.
- Name things his way: the places, and the project names and words in `projects.yaml` (`name`, `says`); no repo slugs, agent names or IDs in what he reads unless there's a link to click. Never add a per-repo project, workspace or space to an app; `wb check --all` fails on one (Daniel, 4 November: "i just want inbox / workbench / personal / netnet / ritual").
- Words: bots are persistent assistants (Grok Bot, Hermes profiles, Muse, Dots, Instinct; Manus's Q later). Agents are workers inside harnesses, from long-lived ones to subagents that live for one task. Say "Grok Bot" when you mean Grok Bot.
- The win: good NetNet videos made with minimal supervision. Creative calls are his; a passing check is not his approval.

## Needs Daniel's yes
Use approval he has already given for the specific action. Otherwise ask before opening accounts, changing keys or permissions, spending money, uploading files outside the approved purpose, deploying, changing live services, deleting anything you can't restore, or sending messages or posts. Approved media providers and budgets: `docs/providers/provider-authority.md`. No yes needed (Daniel, 20 October): installing and setting up software on the Hetzner server or the VMs, giving engines and bots access to Executor, and usage on his subscriptions and provider keys within their caps.

Crypto signing, trades, transfers, token approvals and changes to who controls funds each need his approval of the exact action and limits; no other approval covers them. Don't repeat an action that may spend money or affect someone else until you know whether the first attempt worked. Keep private repos private; never make one public to connect a tool.

## Keys
Load secrets from Keychain (`tools/auth/keychain-env.sh`; names in `docs/providers/auth-env.md`). For secrets held only in 1Password, use `withkeys VAR -- command`, which asks for Touch ID. Never put passwords, seed phrases, private keys or API keys in files, prompts, logs or command arguments. The one exception is a tool's own login file at the path and in the format its own sign-in writes, mode 600 (for example `~/.supermemory-claude/credentials.json`); never in a repo or a note. Keep service logins in Executor rather than adding them separately to each agent app; such tool login files don't count.

## Venture rules (copied from snakeg0d/personal-workspace AGENTS.md)

## Personal

Personal is the place Daniel sees for his own projects (owner snakeg0d); this repo is its hub (venture rules, `STATE.md`, records; model: `~/dev/docs/architecture/MODEL.md`). A chat in Personal coordinates; the work lands in the repo that owns it: open that repo, or your own worktree under `~/dev/worktrees`, and before working in it read its `AGENTS.md`; put branches, issues and pull requests there, then report back in the Personal chat. The hub's own pull-request and worktree buttons act on the hub only.

### What you can ask for here

In Daniel's words (names and words from `~/dev/projects.yaml`; stages: `~/dev/projects/README.md`). Paths are under `/Users/danielgosek/dev/projects/personal/` (Hetzner: `/home/shed/dev/projects/personal/`).

| He says | Project | Where |
|---|---|---|
| vault, obsidian | Vault: the agents' shared memory and his notes (private) | `vault/` (`snakeg0d/vault`) |
| morning paper, the paper | Morning paper: his daily paper, written by his bots and agents | `apps/paper/` (`snakeg0d/paper`) |
| ghostty, ghostty tui, ghostty studio | Ghostty TUI: keyboard-driven config for the Ghostty terminal | `apps/ghostty-tui/` (`snakeg0d/ghostty-tui`) |

- The workbench rules (`~/dev/AGENTS.md`) apply as written; Personal adds nothing beyond this routing.
- The hub's `career/` holds local-only material (ignored); it does not go to GitHub.

Place: Personal › Apps › Ghostty TUI (paused). Daniel says: ghostty, ghostty tui, ghostty studio. Hub: snakeg0d/personal-workspace. New unsorted work: ~/dev/inbox; new project: wb new personal <area> <repo>.
<!-- workbench:end -->
