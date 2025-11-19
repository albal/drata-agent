# GitHub Automation Overview

This directory contains all automation supporting security scanning, dependency hygiene, and upstream sync tasks for `drata-agent`. The sections below describe each workflow, trigger, and any required secrets.

## Security Scanning Workflows

| Name | File | Purpose | Triggers | Notes |
| --- | --- | --- | --- | --- |
| CodeQL JavaScript | `workflows/codeql.yml` | Runs GitHub CodeQL to detect JavaScript/TypeScript vulnerabilities and uploads SARIF results to the Security tab. | Push to `main`, PRs targeting `main`, weekly cron (`02:20` UTC Mondays). | Uses Node `22.14.0` and installs dependencies via `yarn install --frozen-lockfile`. |
| Semgrep SAST | `workflows/semgrep.yml` | Executes the OSS Semgrep CLI against the `p/ci` ruleset and publishes findings to code scanning. | Push to `main`, PRs targeting `main`, weekly cron (`03:05` UTC Mondays). | Optional: add `SEMGREP_APP_TOKEN` secret to surface results in Semgrep Cloud. |
| OSV Scanner | `workflows/osv-scanner.yml` | Runs Google’s OSV dependency scanner on `yarn.lock` and uploads SARIF to GitHub. | Push to `main`, PRs targeting `main`, weekly cron (`04:35` UTC Mondays). | Installs Go 1.22 and the `osv-scanner` CLI at runtime. |

Each workflow emits SARIF artifacts into GitHub Advanced Security so findings appear centrally under **Security ▸ Code scanning alerts**. Adjust cron expressions or branches as needed for different cadences.

## Repository Sync Workflow

| Name | File | Purpose | Triggers | Notes |
| --- | --- | --- | --- | --- |
| Sync Upstream | `workflows/sync-upstream.yml` | Detects drift between `origin/main` and `upstream/main`, fast-forwards the local branch, and preserves the `.github` directory. | Daily cron (`04:00` UTC) and manual `workflow_dispatch`. | Requires the `upstream` remote to point at `git@github.com:drata/drata-agent.git`. Uses `GITHUB_TOKEN` for pushes and backs up `.github` before resetting. |

Manual run tip: open **Actions ▸ Sync Upstream ▸ Run workflow** to trigger an on-demand sync.

## Dependency Automation

`dependabot.yml` manages two ecosystems:

- **npm** (root `package.json` / `yarn.lock`): weekly PRs, up to 10 open, labeled `dependencies`, commits prefixed with `chore(scope)`.
- **GitHub Actions**: weekly update checks for workflow actions, also labeled `dependencies` with the same commit prefix.

## Maintenance Tips

- Review Dependabot PRs promptly; the security workflows rely on current actions and packages.
- To add more scanners, create a dedicated workflow file per tool to keep ownership clear and update this README accordingly.
- If you change branch names or remotes, update both the relevant workflow YAML and the tables here to keep documentation accurate.
