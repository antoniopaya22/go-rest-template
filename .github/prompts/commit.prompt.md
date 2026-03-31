---
name: "commit"
description: "Stage changes, create a Conventional Commit, and push the current branch with safe Git checks"
argument-hint: "Optional commit intent, scope, or issue reference"
agent: "agent"
---

Commit the current work in this repository.

Use the repository workflows already defined in [Makefile](../../Makefile) and [ci.yml](../workflows/ci.yml) when choosing validation commands.

Workflow:

1. Inspect the current branch, `git status --short --branch`, and a diff summary before making changes.
2. If the user supplied extra text with `/commit`, treat it as commit intent, preferred scope, or issue reference.
3. If there are merge conflicts, an in-progress rebase/cherry-pick, or multiple clearly unrelated change sets, stop and ask before committing.
4. If there is nothing to commit, say so and stop.
5. Stage the intended changes with `git add -A`.
6. Write the commit message using Conventional Commits:
   - Format: `<type>(<optional-scope>): <imperative summary>`
   - Choose the most accurate type, such as `feat`, `fix`, `refactor`, `docs`, `test`, `build`, `ci`, or `chore`
   - Keep the subject concise and specific
   - Add a body only when it adds value, for example rationale, validation, or breaking-change context
   - Avoid vague messages such as `update`, `changes`, or `fix stuff`
7. Run sensible validation before committing when the diff warrants it:
   - Prefer the repo workflows, for example `make vet`, `make lint`, `make test`, or the smallest relevant subset
   - If validation fails, stop and report the failure instead of committing
8. Commit non-interactively.
9. Push to the current branch:
   - Use the existing upstream when present
   - If no upstream exists, use `git push -u origin <current-branch>`

Safety rules:

- Never force-push
- Never amend unless the user explicitly asks
- Never include unrelated changes just to make the working tree clean
- Prefer non-interactive git commands only

Report back with:

- current branch name
- final commit message
- short commit hash
- whether validation ran and what passed