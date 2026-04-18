# HAPPY WORK 🙂

> Happy Work 🙂 addresses the most significant bottlenecks in the Software Development Life Cycle (SDLC): context switching and boilerplate overhead. Built as Go service that listens for Jira webhooks, uses Gemini to implement the task, and opens a Bitbucket pull request automatically.

## How it works

```
  Jira issue → "In Progress"
        │
        ▼
  POST /webhook/jira
        │
        ▼
  Gemini generates code changes
        │
        ▼
  Bitbucket: create branch → commit files → open PR
```

## Prerequisites

| Tool | Purpose |
|------|---------|
| Go 1.26+ | Build the service |
| Docker | Container deployment |
| Jira (Cloud or Server) | Webhook source |
| Bitbucket Cloud | Target repo + App Password |
| Google Gemini API key | Gemini access |

---

## Local development

```bash
# 1. Clone and install deps
git clone https://github.com/your-org/happy-work
cd happy-work
go mod download

# 2. Configure env
cp .env.example .env
# Edit .env with your real values

# 3. Export env and run
export $(cat .env | xargs)
go run ./cmd/server

# 4. Test with curl
curl -X POST http://localhost:8080/webhook/jira \
  -H "Content-Type: application/json" \
  -d @testdata/issue_transitioned.json
```

---

## Jira webhook setup

1. Go to **Jira Settings → System → WebHooks → Create WebHook**.
2. Set the URL to `https://<your-service>/webhook/jira`.
3. Under **Issue**, check **updated**.
4. (Optional) Set a secret and put it in `JIRA_WEBHOOK_SECRET`.

---

## Bitbucket API key

1. Go to **Bitbucket → Repository settings → Access tokens** (for a repo-scoped token) or **Workspace settings → Access tokens** (for workspace-wide access).
2. Click **Create access token**, give it a name, and select scopes: `Repositories: Read & Write` and `Pull requests: Read & Write`.
3. Copy the generated token and set it as `BITBUCKET_API_KEY`.

---

## Cloud deployment (Vercel)

happy-work runs as a **Vercel serverless function** — no server to manage.

### 1. Install Vercel CLI & deploy

```bash
npm i -g vercel
vercel login

# From the project root:
vercel --prod
```

Vercel detects `vercel.json`, builds `api/webhook.go` with the Go runtime, and exposes it at:
```
https://<your-project>.vercel.app/webhook/jira
```

### 2. Set environment variables

```bash
vercel env add GEMINI_API_KEY
vercel env add BITBUCKET_WORKSPACE
vercel env add BITBUCKET_USERNAME
vercel env add BITBUCKET_API_KEY
# Optional:
vercel env add JIRA_WEBHOOK_SECRET
vercel env add BITBUCKET_REPO_SLUG
```

Or set them in the Vercel dashboard under **Project → Settings → Environment Variables**.

### 3. Point Jira at your function

Use `https://<your-project>.vercel.app/webhook/jira` as the Jira webhook URL.

> **Note on timeouts:** Vercel Hobby plan limits function execution to **10 seconds**. Since Gemini + Bitbucket operations can take longer, use the **Pro plan** (60s limit) or add a queue (e.g. Vercel KV + background job) for reliability.

---

## Environment variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `PORT` | No | `8080` | HTTP port |
| `JIRA_TRIGGER_STATUS` | No | `In Progress` | Transition status to react to |
| `JIRA_WEBHOOK_SECRET` | No | _(empty)_ | HMAC secret for payload verification |
| `GEMINI_API_KEY` | **Yes** | – | Google Gemini API key |
| `GEMINI_MODEL` | No | `gemini-2.0-flash` | Gemini model ID |
| `BITBUCKET_BASE_URL` | No | `https://api.bitbucket.org/2.0` | Bitbucket API base |
| `BITBUCKET_WORKSPACE` | **Yes** | – | Bitbucket workspace slug |
| `BITBUCKET_API_KEY` | **Yes** | – | Bitbucket repository/workspace access token |
| `BITBUCKET_REPO_SLUG` | No | _(derived from Jira project key)_ | Override repo slug |

---

## Project structure

```
happy-work/
├── api/
│   └── webhook.go      # Vercel serverless entry point
├── cmd/server/         # local dev entry point (wraps api/webhook.go)
├── config/             # env var loading & validation
├── webhook/            # Jira payload parsing & verification
├── gemini/             # Google Gemini API client
├── vercel.json         # Vercel routes & build config
├── Dockerfile          # optional: for local Docker testing
├── .env.example
└── README.md
```

---

## Health check

`GET /healthz` → `200 OK` — used by load balancers and container orchestrators.