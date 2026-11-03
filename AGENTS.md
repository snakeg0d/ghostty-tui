<!-- workbench:begin (managed by ~/dev/tools/workbench/sync-rules; edit the sources, not this block) -->
Venture: Personal (personal in snakeg0d/dev projects.yaml). Project: Personal · Ghostty TUI (snakeg0d/ghostty-tui, paused). Hub: snakeg0d/personal-workspace. Ritual: no.

## Workbench rules (copied from snakeg0d/dev AGENTS.md)

## How Daniel works
- He says what he wants in plain words, often from his phone. Use the tools that already exist, then reply briefly: what changed and where, plus anything that failed or needs his decision.
- He approves in words. Do reversible work within his request without asking, except the actions under "Needs Daniel's yes".
- Talk like a colleague, in plain words: no nicknames, no fixed report formats, no filler.
- Words: bots are persistent assistants (Grok Bot, Hermes profiles, Muse, Dots, Instinct; Manus's Q later). Agents are workers inside harnesses, from long-lived ones to subagents that live for one task. Say "Grok Bot" when you mean Grok Bot.
- The win: good NetNet videos made with minimal supervision. Creative calls are his; a passing check is not his approval.

## Needs Daniel's yes
Use approval he has already given for the specific action. Otherwise ask before opening accounts, changing keys or permissions, spending money, uploading files outside the approved purpose, deploying, changing live services, deleting anything you can't restore, or sending messages or posts. Approved media providers and budgets: `docs/providers/provider-authority.md`. No yes needed (Daniel, 20 October): installing and setting up software on the Hetzner server or the VMs, giving engines and bots access to Executor, and usage on his subscriptions and provider keys within their caps.

Crypto signing, trades, transfers, token approvals and changes to who controls funds each need his approval of the exact action and limits; no other approval covers them. Don't repeat an action that may spend money or affect someone else until you know whether the first attempt worked. Keep private repos private; never make one public to connect a tool.

## Keys
Load secrets from Keychain (`tools/auth/keychain-env.sh`; names in `docs/providers/auth-env.md`). For secrets held only in 1Password, use `withkeys VAR -- command`, which asks for Touch ID. Never put passwords, seed phrases, private keys or API keys in files, prompts, logs or command arguments. The one exception is a tool's own login file at the path and in the format its own sign-in writes, mode 600 (for example `~/.supermemory-claude/credentials.json`); never in a repo or a note. Keep service logins in Executor rather than adding them separately to each agent app; such tool login files don't count.

## Venture rules (copied from snakeg0d/personal-workspace AGENTS.md)

## Personal

The Personal venture: Daniel's own projects, owner snakeg0d. This repo is its hub: venture rules, `STATE.md` and records. Projects: `vault` (shared agent memory, knowledge), `paper` (The Morning Paper, paused), `ghostty-tui` (paused). Registry: `~/dev/projects.yaml`; model: `~/dev/docs/architecture/MODEL.md`.

- The workbench rules apply as written; nothing venture-specific yet.
- `career/` holds local-only material (ignored); it does not go to GitHub.

Place: Personal › Apps › Ghostty TUI (paused). Hub: snakeg0d/personal-workspace. New unsorted work: ~/dev/inbox; new project: wb new personal <area> <repo>.
<!-- workbench:end -->
