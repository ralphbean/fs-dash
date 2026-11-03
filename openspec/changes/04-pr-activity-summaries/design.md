## Context

This optional feature consumes the detail snapshots from change 03. [Official OpenAI model documentation](https://developers.openai.com/api/docs/models) identifies `gpt-6-luna` as the Luna model ID, and [Codex integration guidance](https://developers.openai.com/blog/codex-as-a-platform) describes `codex exec` for bounded background jobs. The installed Codex CLI and account access must be checked during implementation. See proposal.md and the summary spec.

## Goals / Non-Goals

**Goals:** Produce a short, sourced update digest without slowing factual details, and avoid recomputing unchanged inputs.

**Non-Goals:** Autonomous PR actions, a persistent chat session, or using generated text to decide merge readiness.

## Decisions

1. **Optional bounded subprocess.** Use a `Summarizer` port and a local `codex exec` adapter with configurable executable/model (default model ID `gpt-6-luna`), timeout, output limit, and one in-flight job per PR fingerprint. Pass a bounded factual snapshot as input and disable workspace edits for this task. A long-lived app-server session was considered but adds lifecycle and approval complexity for a one-shot digest.
2. **Fingerprint the actual evidence.** Hash source host/repo/PR, head SHA, review/comment IDs and update times, check rollup version, prompt version, and model setting. Reuse only an exact cache hit. Do not key solely by head SHA, because comments and checks change without commits.
3. **Cache is local and attributable.** Store summary text, fingerprint, source snapshot time, model ID, generation time, and failure metadata in the XDG cache path. Show a stale prior result while regenerating, clearly labeled. The factual detail panel remains authoritative.
4. **Limit content exposure.** Include only configured PR data and URLs in the prompt, omit credentials and private local files, and document that enabled generation sends the supplied content through the user's Codex configuration. Do not include arbitrary repository files by default.

## Risks / Trade-offs

- Codex CLI/model availability differs by machine or account → optional feature, startup capability check, and visible error.
- Summary can omit or misstate facts → label generated output and link to source evidence in details.
- Input state can change during generation → bind result to fingerprint and discard it as current if inputs changed.
- Cost and latency can grow → opt-in default, bounded input/output, coalescing, and cache reuse.

## Migration Plan

Feature is disabled by default; no migration. Cache can be deleted without affecting PR data.
