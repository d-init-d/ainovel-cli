# D Research Plugin Integration

This fork ships with a bundled d-research bridge for research-first story planning. It is intentionally not a generic plugin marketplace yet; the goal is to keep d-research packaged cleanly today while leaving a path to a broader plugin system later.

## What is included

- `plugins/d-research/`: a thin plugin manifest and workflow notes.
- `internal/research/`: the Go bridge that resolves the plugin and runs d-research Node.js scripts.
- `research_brief`: an Architect-only tool for web research before foundation planning.
- `research_pack`: a compact report injected into `novel_context` for Architect and Writer.

## Enable it

Add this to your `~/.ainovel/config.json` or project `.ainovel/config.json`:

```jsonc
"research": {
  "enabled": true,
  "plugin": "d-research",
  "auto": true,
  "max_queries": 8,
  "max_results_per_query": 6,
  "max_sources": 12,
  "timeout_seconds": 120,
  "browser": {
    "enabled": true,
    "headless": true,
    "timeout_seconds": 30,
    "extract": true
  }
}
```

If `enabled` is false or omitted, research is disabled. Missing Node.js or Playwright will not break normal novel writing.

## TUI command

Use `/d-research` either as the first prompt (research-first startup) or while a project is open:

```text
/d-research fusion torchship constraints freshness=2025-01-01 max=8
```

You can attach local research notes by passing file paths:

```text
/d-research fusion torchship constraints file="D:\Docs\fusion notes.md" url=https://example.com/paper
```

The command asks the Coordinator to dispatch an Architect agent, and the Architect calls `research_brief`. Local files become `local_file` evidence in the report and are included in the compact `research_pack`.

This is path-based attachment in the terminal, not a GUI upload widget. Local research attachments accept TXT, Markdown, and DOCX; DOCX text is extracted locally with bounded ZIP/XML processing. Legacy `.doc`, PDF, and EPUB are not accepted by this workflow.

Privacy note: excerpts from attached files can be included in prompts sent to the configured model provider. Do not attach confidential material unless that provider and deployment are approved to receive it. Source content is treated as untrusted evidence and cannot override agent/system instructions.

For importing an existing source novel, use the existing import flow instead:

```text
/import "D:\Books\source-novel.txt"
```

`/import` currently supports text/markdown source novels. It reconstructs foundation/chapter state so the agent can continue from or subsequently revise the imported book; it does not automatically rewrite the entire book in one command. It is a path-based TUI command, not a GUI drag-and-drop upload widget.

## Prompt attachments versus novel import

Use `/attach` when a file is reference material for the next user prompt and must not be interpreted as a novel to import or rewrite:

```text
/attach "D:\Docs\world rules.docx" "D:\Docs\science notes.md"
Hãy dùng các tài liệu đính kèm làm knowledge để kiểm tra logic chương 4.
```

Attachments are one-shot: after the next normal prompt is sent, the queue is cleared. Use `/attachments` to inspect the queue and `/detach` to clear it. Large documents are locally extracted and reduced to query-relevant excerpts before being placed in the prompt. `/d-research` also consumes queued attachments as explicit `file_paths`.

`/import` retains its original first-party meaning and is not used for knowledge attachments.

## Plugin resolution

The bridge resolves d-research in this order:

1. Explicit `research.plugin_path` when configured. This path is authoritative; if it is wrong, the bridge reports the problem instead of silently falling back.
2. Current-working-directory `plugins/d-research/`.
3. Executable-adjacent `plugins/d-research/` (important for native release archives).
4. `%USERPROFILE%\.codex\skills\d-research`.
5. `%USERPROFILE%\.agents\skills\d-research`.

The repo-local bundle includes the core d-research scripts used by ainovel-cli. If a future lightweight bundle has no scripts but a full d-research skill is installed under `.codex` or `.agents`, the bridge can borrow those scripts.

When distributing a native executable, ship the `plugins/d-research` directory next to the executable using the same relative layout; the Node scripts cannot run from inside the Go binary itself.

## Runtime dependencies

Node.js 18+ is required for script execution.

Playwright is required for browser probing and extraction. Install it in the bundled plugin directory so Node can resolve it from the scripts:

```powershell
cd "D:\Downloads\fork ainovel-cli\plugins\d-research"
npm install
npx playwright install chromium
```

If Playwright is unavailable, the tool can still preserve search-result evidence, but browser probe/extract evidence will be limited.

## Output files

Reports are saved under:

```text
<output>/meta/research/<report-id>/
```

Files:

- `report.json`: full report with sources, blockers, contradictions, and file links.
- `evidence-ledger.json`: source evidence ledger.
- `compact.json`: small context-safe report used by `novel_context`.

The latest pointer is saved at:

```text
<output>/meta/research/latest.json
```

## Safety boundary

D Research is read-only and lawful-access only:

- Do not bypass login walls, paywalls, authentication, CAPTCHA, bot detection, or rate limits.
- Do not access non-public or restricted content.
- Use public pages, public APIs, official docs, papers, datasets, and other lawful sources.
- When blocked, record the blocker and move to alternative sources.
