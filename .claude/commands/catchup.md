---
description: Read all changed files in the current git branch to catch up on context
---

Read all files that have been changed in the current git branch to understand the recent work.

Steps:
1. Use `git diff --name-only main..HEAD` to find all changed files
2. Read each changed file using the Read tool
3. Provide a brief summary of what has changed in the project

If not in a git repository, read all files in backend/, proxy/, and frontend/ directories instead.
