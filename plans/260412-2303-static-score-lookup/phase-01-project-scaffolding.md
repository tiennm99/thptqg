# Phase 01 — Project Scaffolding

## Context Links
- [plan.md](./plan.md)
- [Converter.java](../../src/main/java/dev/miti99/thptqg2017/Converter.java) — reference parsing logic
- [Student.java](../../src/main/java/dev/miti99/thptqg2017/entity/Student.java) — reference data model

## Overview
- **Priority**: P1 (blocker for all other phases)
- **Status**: Pending
- **Description**: Initialize Node.js project, Vite+React app, and folder structure

## Key Decisions

- **Monorepo with single package.json** at root — project is small, no need for workspaces
- **better-sqlite3** for build script (fast native writes), **sql.js** for browser runtime
- **Vite + React + TypeScript** for the static site
- Keep Java source intact — no deletions in this phase

## Architecture

```
thptqg2017/
├── scripts/                    # Node.js build scripts
│   └── build-database.js       # Excel -> SQLite converter
├── web/                        # Vite + React app
│   ├── src/
│   │   ├── main.tsx
│   │   ├── app.tsx
│   │   └── components/
│   ├── public/                 # .db file goes here after build
│   ├── index.html
│   └── vite.config.ts
├── package.json                # Root: scripts + web deps
├── src/main/resources/raw/     # Existing Excel files (unchanged)
└── ... (existing Java files unchanged)
```

## Related Code Files

**Create:**
- `package.json` — root package with workspaces or flat deps
- `web/index.html` — Vite entry
- `web/vite.config.ts`
- `web/src/main.tsx`
- `web/src/app.tsx`
- `web/tsconfig.json`
- `scripts/` directory (empty, placeholder)

**No modifications to existing files.**

## Implementation Steps

1. Initialize `package.json` at project root
   ```bash
   npm init -y
   ```
2. Install build-script dependencies
   ```bash
   npm install --save-dev better-sqlite3 xlsx
   ```
3. Install web dependencies
   ```bash
   npm install react react-dom sql.js
   npm install --save-dev @types/react @types/react-dom typescript vite @vitejs/plugin-react
   ```
4. Create `web/` directory with Vite scaffold
   - `web/index.html` — minimal HTML shell
   - `web/vite.config.ts` — configure root, public dir, build output
   - `web/src/main.tsx` — React entry
   - `web/src/app.tsx` — placeholder App component
   - `web/tsconfig.json` — strict TS config
5. Create `scripts/` directory
6. Add npm scripts to `package.json`:
   ```json
   {
     "scripts": {
       "build:db": "node scripts/build-database.js",
       "dev": "vite --config web/vite.config.ts",
       "build": "vite build --config web/vite.config.ts",
       "preview": "vite preview --config web/vite.config.ts"
     }
   }
   ```
7. Update `.gitignore` to add:
   ```
   node_modules/
   web/dist/
   web/public/thptqg2017.db
   ```
8. Verify `npm run dev` starts without errors

## Todo List

- [ ] Create package.json with all dependencies
- [ ] Create web/ directory with Vite + React scaffold
- [ ] Create scripts/ directory
- [ ] Update .gitignore
- [ ] Verify `npm run dev` starts clean

## Success Criteria

- `npm install` completes without errors
- `npm run dev` opens a blank React page in browser
- No existing files modified (Java code untouched)
- All new files use kebab-case naming

## Risk Assessment

| Risk | Mitigation |
|------|------------|
| better-sqlite3 native build fails on Windows | Use prebuild binaries (default); fallback to sql.js for build script too |
| Vite config path issues with nested web/ dir | Set `root: 'web'` in vite.config.ts explicitly |

## Next Steps
- Phase 02 depends on `scripts/` dir and `better-sqlite3` being available
- Phase 03 depends on `web/` scaffold and React being configured
