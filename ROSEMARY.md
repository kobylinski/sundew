# Rosemary team for Sundew

Optional. Work directly with the user by default; only an explicit instruction to join activates
the team ("Join as the Programmer; read ROSEMARY.md and your contract"). [AGENTS.md](AGENTS.md) is
the shared starting point.

A small, consolidated team: four contracts, one agent each.

| Contract | Keeps | Model, effort |
| --- | --- | --- |
| [Integrator](docs/contracts/integrator.md) — Architect and Integrator in one | `explain`, `decide`, `build` | top tier, medium |
| [Programmer](docs/contracts/programmer.md) — Task Manager and Programmer in one | raises all gates | strong coder, high |
| [UX Designer](docs/contracts/designer.md) | `design` | top tier, high |
| [Reviewer](docs/contracts/reviewer.md) — Code Reviewer and Tester in one | `review_code`, `review_tests` | top tier, high |

Names: [agent-name skill](.claude/skills/agent-name/SKILL.md). Tasks: `tasks:` (git-native,
`rosemary.yml`). Decisions: `docs/journal/YYYY-MM-DD/decision-<slug>.md`. Product direction:
[docs/BRIEF.md](docs/BRIEF.md).

## The team

Marek is the operator. Four contracts, each held by one agent at a time; a name is a seat
(first letter = role, from the [agent-name skill](../../.claude/skills/agent-name/SKILL.md)).

| Contract | Keeps gates | The job |
| --- | --- | --- |
| Integrator | `explain`, `decide`, `build` | Talks with the operator; writes and assigns tasks; is the only way into `main`. |
| Programmer | raises them all | Owns one task end to end: lane, code, tests, proof, rebase, close. |
| UX Designer | `design` | Prototypes the UI the operator accepts; hands over the README a Programmer builds from. |
| Reviewer | `review_code`, `review_tests` | Reads the diff and judges it; judges the tests; writes end-to-end tests. |

Tasks are `tasks:<id>` (git-native, shared through Rosemary ticket refs). Nothing deploys from a
push; a release is a tagged image.

## How a task moves

1. Operator and Integrator talk; the Integrator writes the task (Outcome, Scope, How to verify,
   Route, Decided and open, Where to look) and binds it to the Programmer.
2. The Programmer cuts the lane — branch `task/<id8>-<slug>`, worktree `.worktrees/<id8>-<slug>`
   — and writes Requirements and Verification on the task before any code.
3. Gates in the Route's order: `design` (Designer) when a screen is not decided; the Programmer
   writes the code and tests, then raises `review_code` and `review_tests` (Reviewer); the Reviewer
   also writes or extends the end-to-end acceptance when the Route asks.
4. The Programmer proves it — `go vet`, `go test ./...`, the image builds, the acceptance script
   passes against the image — rebases on `origin/main`, proves again, raises `build`.
5. The Integrator merges with `--no-ff`, runs the same proof on the merge commit, pushes `main`,
   passes `build` with `integrated=<merge commit>`.
6. The Programmer removes the lane, writes the Summary, closes the task, tells the Integrator it is
   free.

`explain` (a question about the brief) and `decide` (a choice the brief does not settle) go to the
Integrator; what only the operator can answer becomes an operator question on the task.

Where Rosemary refuses something a contract asks for, report the refusal in its own words; do not
work around it.
