# Forgejo Test Effectiveness & Security Evaluator

An automated, AI-powered **Forgejo Action** that analyzes Pull Requests and branches/refs to evaluate test suite effectiveness, security surface coverage, and code clarity. Powered by **Gemini 2.5 Pro**, this action provides deep code review insights, tracks state across iterations using **Git Notes**, and posts structured recommendations directly as PR comments or workflow log outputs.

---

## Architecture Overview
+-----------------------------------------------------------------------+
|                         Forgejo Action Runner                         |
|                                                                       |
|  1. Context & Diff Fetcher  <--->  Forgejo API (PR Files & Comments)  |
|            |                                                          |
|            v                                                          |
|  2. Git Notes State Reader (Fetch refs/notes/ai-test-bot for SHA)     |
|            |                                                          |
|            v                                                          |
|  3. Source/Test Splitter & Context Assembler                          |
|            |                                                          |
|            v                                                          |
|  4. Gemini Evaluator Engine (Large Context Model)                     |
|            |                                                          |
|            v                                                          |
|  5. Delta Engine (Compare Current Evaluation vs. Prior SHA Note)      |
|            |                                                          |
|            v                                                          |
|  6. Git Notes Writer (Attach JSON State to Head Commit & Push Ref)    |
|            |                                                          |
|            v                                                          |
|  7. Forgejo API Publisher (Post/Update PR Summary Comment)            |
+-----------------------------------------------------------------------+
---

## Key Features

* **Test Effectiveness Categorization (Levels 0–4):** Distinguishes between performative/trivial tests and robust, security-critical assertions.
* **Security Surface Mapping:** Identifies sensitive code changes (auth, input validation, cryptographic boundaries) and checks for corresponding negative/boundary test cases.
* **Git-Native Persistence via Git Notes:** Stores evaluation metadata cleanly in `refs/notes/ai-test-bot` attached to commit SHAs—no cluttering PR threads with hidden HTML comments.
* **Delta Tracking:** Tracks resolution state (`[RESOLVED]`, `[PARTIALLY ADDRESSED]`, `[UNRESOLVED]`) across PR update pushes (`synchronize` events) and branch commits.
* **PR and Branch Scan Modes:** Runs seamlessly on Pull Requests or standalone branch/ref pushes and `workflow_dispatch` events without requiring an open PR.
* **Configurable Logging:** Supports configurable log levels (`none`, `info`, `debug`) to output process milestones and step details directly to action logs.
* **Hybrid Prompt Loading:** Uses a built-in default system prompt (`//go:embed`) with support for repository-level prompt overrides (`--prompt-file`).

---

## Evaluation Rubric

### Test Quality & Effectiveness Scale

| Tier | Category | Description |
| :--- | :--- | :--- |
| **Level 0** | **No / Fragile Coverage** | Missing tests, broken tests, or tests with no meaningful assertions. |
| **Level 1** | **Performative / Trivial** | Tests basic getters/setters, string formatting, or mocked implementation details without testing behavior. |
| **Level 2** | **Basic Happy-Path** | Tests standard input/output paths under ideal conditions without boundary or edge-case handling. |
| **Level 3** | **Robust Functional** | Tests complex business logic, error branches, invalid input handling, and boundary conditions. |
| **Level 4** | **Regressive & Security-Critical** | Tests explicit vulnerability mitigations (SQLi, IDOR, race conditions) or uses generative/fuzz inputs to prevent critical system failure. |

---

## Directory Structure

```text
.
├── cmd/
│   └── forgejo-test-evaluator/
│       ├── main.go               # Application entry point & CLI flag parsing
│       └── prompts/
│           └── system_instruction.md # Default embedded system prompt
├── pkg/
│   ├── client/
│   │   ├── forgejo.go            # Forgejo REST API wrapper
│   │   └── gemini.go             # Gemini API client wrapper
│   ├── evaluator/
│   │   ├── analyzer.go           # Context assembler & prompt processor
│   │   └── delta.go              # State comparison engine across commits
│   ├── gitnotes/
│   │   └── notes.go              # Git Notes reader/writer (refs/notes/ai-test-bot)
│   ├── logger/
│   │   └── logger.go             # Log level logging manager (none, info, debug)
│   └── models/
│       └── state.go              # JSON state data structures
├── action.yml                    # Forgejo / Gitea Action metadata definition
├── go.mod                        # Go module definition
└── README.md

## Usage

Add the action to your repository workflow at `.forgejo/workflows/test-evaluator.yml`:

```yaml
name: Test Effectiveness & Security Evaluator

on:
  pull_request:
    types: [opened, synchronize, reopened]
  push:
    branches: [main, develop]
  workflow_dispatch:

jobs:
  evaluate-tests:
    runs-on: docker
    steps:
      - name: Checkout Code
        uses: actions/checkout@v4
        with:
          # Required to fetch full history for Git Notes lookups
          fetch-depth: 0

      - name: Run Test Evaluator Action
        uses: ./.
        with:
          gemini_api_key: ${{ secrets.GEMINI_API_KEY }}
          forgejo_token: ${{ secrets.GITHUB_TOKEN }} # or secrets.FORGEJO_TOKEN
          log_level: info
          # Optional ref or branch override when not running in PR context:
          # ref: ${{ github.ref }}
          # Optional custom prompt override:
          # custom_prompt_path: ".forgejo/prompts/custom_rules.md"
```
