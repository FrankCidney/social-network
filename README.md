# Social Network Platform

Full-stack social network built with Go, Next.js, WebSockets, and SQLite.

## Stack

- Backend: Go
- Frontend: Next.js + TypeScript
- Database: SQLite
- Realtime: WebSockets

## Run locally

1. Start the backend:
   ```bash
   cd backend
   go run ./cmd
   ```
2. Start the frontend in a second terminal:
   ```bash
   cd frontend
   npm install
   npm run dev
   ```

App: `http://localhost:3000`  
API: `http://localhost:8080`

If needed, set `frontend/.env.local` to:

```env
NEXT_PUBLIC_API_URL=http://localhost:8080
```

## Run with Docker

From the repo root:

```bash
NEXT_PUBLIC_API_URL=http://localhost:8080 docker compose up --build
```

App: `http://localhost:3000`  
API: `http://localhost:8080`
