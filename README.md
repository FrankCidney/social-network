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

The frontend defaults to `http://localhost:8080` for API requests in local development. You only need `frontend/.env.local` if you want a different backend URL:

```env
NEXT_PUBLIC_API_URL=http://localhost:8080
```

## Run with Docker

From the repo root:

```bash
docker compose up --build
```

App: `http://localhost:3000`  
API: `http://localhost:8080`

This starts the frontend and backend containers together. The backend stores its SQLite database in the Compose-managed `sqlite_data` volume.

If the frontend needs to talk to a backend on a different host, override the public API URL when building:

```bash
NEXT_PUBLIC_API_URL=http://your-backend-host:8080 docker compose up --build
```
