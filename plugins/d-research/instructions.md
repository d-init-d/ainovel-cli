# D Research Workflow Instructions

## Overview

D Research is a browser-first deep research and lawful public-data collection workflow for AI agents. It performs research using web search, browser probing, extraction, evidence ledgers, contradiction checks, and blocker reports.

## Research workflow

1. Define the question.
   - Start with a clear research goal.
   - Split vague requests into answerable sub-questions.

2. Decompose the topic.
   - Core facts and definitions.
   - Official or primary sources.
   - Scientific and technical details.
   - Contradictions, limitations, and caveats.
   - Recent developments when freshness matters.

3. Map source basins.
   - Official documentation and standards bodies.
   - Academic papers and preprints.
   - Technical blogs and engineering reports.
   - Government or regulatory sources.
   - Industry analysis and public datasets.

4. Fan out queries.
   - Use several queries targeting different source basins.
   - Prefer specific domain terms over generic search terms.

5. Probe with a browser.
   - Check whether candidate pages are reachable.
   - Detect paywalls, login walls, CAPTCHA, bot detection, rate limits, and broken pages.
   - Record access status and blockers.

6. Extract accessible data.
   - Preserve source URL, title, timestamp, access method, and confidence hint.
   - Keep useful snippets and extracted text when lawful and accessible.

7. Build the evidence ledger.
   - Store source facts, snippets, access status, source type, and blocker details.
   - Mark contradiction candidates found through limitation or criticism queries.

8. Run a contradiction pass.
   - Include limitation, criticism, controversy, and debate queries.
   - Record unresolved contradictions instead of inventing a resolution.

9. Report blockers.
   - Paywall, login required, CAPTCHA, bot challenge, rate limit, timeout, broken page, or missing dependency.

10. Synthesize.
   - Produce a compact report with evidence highlights, caveats, contradictions, blockers, and confidence notes.

## Safety boundary

D Research is read-only and lawful-access only.

- Do not bypass login walls, paywalls, authentication, CAPTCHA, bot detection, or rate limits.
- Do not access non-public or restricted content.
- Do not scrape personal data unless it is lawfully public and relevant.
- Use public web pages, public APIs, official docs, papers, datasets, and archives.
- When blocked, record the blocker and move to alternative sources.
