---
description: Prepare a clean pull request with testing and formatting
---

Prepare the code for a pull request by running tests, formatting, and reviewing changes.

Steps:
1. Format all Go code: `go fmt ./...`
2. Run all tests: `go test ./...`
3. If tests fail, fix the issues before continuing
4. Run `git status` to see all changes
5. Run `git diff` to show a summary of modifications
6. Stage all changes: `git add .`
7. Show final git status
8. Ask the user for confirmation before creating a commit

Do NOT automatically commit. Wait for user's commit message and approval.
