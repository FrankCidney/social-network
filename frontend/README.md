# Social Network Frontend

Next.js frontend for the social network app.

## Local environment

Create `frontend/.env.local` from `frontend/.env.example`:

```bash
cp .env.example .env.local
```

For local development with the backend on port `8080`, use:

```bash
NEXT_PUBLIC_API_URL=http://localhost:8080
```

Restart `npm run dev` after changing this value because Next.js reads public env vars into the browser bundle.

## Development

```bash
npm install
npm run dev
```

The frontend runs at `http://localhost:3000` by default and sends API requests to `NEXT_PUBLIC_API_URL`.
