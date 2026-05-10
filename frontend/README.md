# Blackgrid Frontend

React + TypeScript + Vite application for the Blackgrid IPAM/monitoring UI.

## Package manager

This project uses **npm** exclusively. The `package-lock.json` is the
authoritative lockfile and is checked in. Do not commit `pnpm-lock.yaml`,
`yarn.lock`, or `bun.lockb` — multiple lockfiles drift apart and produce
non-reproducible builds.

The Docker image (`frontend/Dockerfile`) runs `npm install && npm run build`,
so any contributor change must work with that exact toolchain.

## Build from a clean checkout

```bash
cd frontend
rm -rf node_modules dist
npm install        # respects package-lock.json
npm run build      # → dist/
```

CI should run `npm ci` instead of `npm install` to fail loudly if the
lockfile is out of date with `package.json`.

## Dev server

```bash
npm run dev
```

The dev server listens on `http://localhost:5173` and the backend's
`CORS_ALLOWED_ORIGINS` default already allows that origin, with
`CORS_ALLOW_CREDENTIALS=true` so the session cookie flows.

## Type-check + lint

```bash
npm run lint
npx tsc -b --noEmit   # type-only check
```

The Docker build (`npm run build`) runs `tsc -b && vite build`, so type
errors block the production build.

## Why not pnpm?

Earlier iterations of this repo had both `package-lock.json` and
`pnpm-lock.yaml` checked in. They drifted, and `npm install` silently
picked up dependency versions that did not match `pnpm install` resolution.
Pick one and stick with it; we picked npm because the deployment Dockerfile
already uses it. If you switch package managers in the future, delete the
old lockfile in the same commit.
