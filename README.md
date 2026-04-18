# happy-work

> A Go service that listens for Jira webhooks, uses Claude to implement the task, and opens a Bitbucket pull request automatically.

## How it works

```
Jira issue → "In Progress"
        │
        ▼
  POST /webhook/jira
        │
        ▼
  Claude generates code changes
        │
        ▼
  Bitbucket: create branch → commit files → open PR
```

## Prerequisites

| Tool | Purpose |
|------|---------|
| Go 1.26.2 | Build the service |
| Docker | Container deployment |
| Jira (Cloud or Server) | Webhook source |
| Bitbucket Cloud | Target repo + App Password |
| Anthropic API key | Claude access |

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

## Bitbucket App Password

1. Go to **Bitbucket → Personal settings → App passwords**.
2. Create a password with scopes: `Repositories: Read & Write`, `Pull requests: Read & Write`.
3. Set `BITBUCKET_APP_PASSWORD` to the generated value.

---

## Cloud deployment (AWS)

### Option A – AWS ECS Fargate

Use the provided `Dockerfile`. Set all environment variables as ECS task environment variables or (recommended) store secrets in **AWS Secrets Manager** / **Parameter Store** and inject them at runtime.

### Option C – GCP Cloud Run

```bash
gcloud run deploy happy-work \
  --image gcr.io/<project>/happy-work:latest \
  --platform managed \
  --region us-central1 \
  --set-env-vars ANTHROPIC_API_KEY=...,BITBUCKET_WORKSPACE=...
```

---

## Environment variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `PORT` | No | `8080` | HTTP port |
| `JIRA_TRIGGER_STATUS` | No | `In Progress` | Transition status to react to |
| `JIRA_WEBHOOK_SECRET` | No | _(empty)_ | HMAC secret for payload verification |
| `ANTHROPIC_API_KEY` | **Yes** | – | Anthropic API key |
| `CLAUDE_MODEL` | No | `claude-sonnet-4-20250514` | Claude model ID |
| `BITBUCKET_BASE_URL` | No | `https://api.bitbucket.org/2.0` | Bitbucket API base |
| `BITBUCKET_WORKSPACE` | **Yes** | – | Bitbucket workspace slug |
| `BITBUCKET_USERNAME` | **Yes** | – | Bitbucket username |
| `BITBUCKET_APP_PASSWORD` | **Yes** | – | Bitbucket app password |
| `BITBUCKET_REPO_SLUG` | No | _(derived from Jira project key)_ | Override repo slug |

---

## Project structure

```
happy-work/
├── cmd/server/         # main entrypoint
├── internal/
│   ├── config/         # env var loading & validation
│   ├── webhook/        # Jira payload parsing & verification
│   ├── claude/         # Anthropic API client
│   └── bitbucket/      # Bitbucket REST API client
├── Dockerfile
├── .env.example
└── README.md
```

---

## Health check

`GET /healthz` → `200 OK` — used by load balancers and container orchestrators.
