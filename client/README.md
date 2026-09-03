# Dagr client

Electron + Vite + React + TypeScript UI for Dagr, styled with [shadcn/ui](https://ui.shadcn.com) and integrated with [shadcnblocks](https://www.shadcnblocks.com). The same UI runs in the browser (`pnpm dev:web` / the nginx image).

## Setup

```bash
pnpm install
cp .env.example .env   # add SHADCNBLOCKS_API_KEY
pnpm dev
```

From the monorepo root: `make client-install` then `make client-dev`.

## Scripts

| Script | Description |
| --- | --- |
| `pnpm dev` | Start Electron in development (instance 1) |
| `pnpm dev:2` | Second Electron against the running Vite server (`DAGR_INSTANCE=2`) |
| `pnpm dev:web` | Vite only (`--mode web`, no Electron window) |
| `pnpm build` | Typecheck and build the Electron renderer |
| `pnpm build:web` | Typecheck and build the browser SPA into `dist/` |
| `pnpm typecheck` | TypeScript only |

From the monorepo root: `make client-dev` then `make client-dev-2`. For the browser: `make client-dev-web`.

## Dual clients (local multi-user)

Each Electron process uses `DAGR_INSTANCE` (default `1`) to set a separate `userData` path (`…/dagr-instance-N`), so sessions do not share `sessionStorage`. Instance `1` keeps the single-instance lock; other instances skip it. Window titles show `Dagr (instance N)` when `N` is not `1`.

Typical flow:

1. Start API + worker (`make compose-infra`, `make migrate-up`, `make run-watch`, `make worker-run`).
2. `make client-dev` (owner login).
3. `make client-dev-2` (second user login) to exercise invites, private channels, message polling, and notifications.

Mention someone with `@handle` (workspace unique). Display names still match as a fallback. Account notification level (Edit profile) is the ceiling for whether notifications are created; per-channel level under Details → Settings can only reduce it. Unread notifications poll while signed in. New unread items also raise a macOS/Windows notification (the bell badge updates even if the OS banner is blocked).

### macOS OS notifications (Electron 42+)

Electron 42 uses Apple’s `UNNotification` API, which rejects the stock `linker-signed` Electron binary. After `pnpm install`, `postinstall` re-signs `Electron.app` (ad-hoc, or with an `Electron Dev` identity if present).

If banners still do not appear:

1. Quit every Dagr/Electron window, then run `pnpm sign:dev` in `client/` (installs `build/icon.png` as the Dock icon and prepares `Dagr.app` so the Dock shows **Dagr**, not Electron).
2. Restart with `make client-dev` (and `make client-dev-2` if needed). If an old **Electron** Dock icon remains, run `killall Dock` once.
3. Trigger a **new** notification after restart (existing unread only update the badge).
4. In **System Settings → Notifications**, allow alerts for **Electron** (or **Dagr** once packaged).
5. Optional stable identity (fewer Keychain prompts after reinstalls): in Keychain Access create a self-signed **Code Signing** certificate named `Electron Dev`, then run `pnpm sign:dev` again.
6. Watch the Electron main-process terminal for `[dagr] OS notification failed:` if delivery is still refused.

## shadcn

- Config: [`components.json`](components.json)
- Registries: `@shadcn` (built-in) and `@shadcnblocks` (Pro key required)
- Brand primary: `rgb(242, 103, 34)`

Cursor MCP for this project is configured at [`.cursor/mcp.json`](../.cursor/mcp.json) with `--cwd client`. Enable it in Cursor Settings so agents can search and add registry items.

## Layout

```text
electron/main      Electron main process
electron/preload   Preload bridge
build/             App icon for packaging (electron-builder)
public/            Static assets copied into the renderer
src/               React renderer (auth screens, chat shell, shadcn UI)
```

### App icon

**Source file to edit:** [`build/icon.png`](build/icon.png)

| Spec | Value |
| --- | --- |
| Path | `client/build/icon.png` |
| Size | **1024×1024** PNG |
| Colour | sRGB, no exotic profiles |
| Shape | Full square canvas. macOS applies the squircle mask itself. |

Design tips for Dock (so it does not look larger than Safari):

- Keep the glyph inside roughly the centre **80%** of the canvas. Leave transparent or matching margin around the edges.
- Prefer a **transparent** background outside the squircle content, or a filled rounded square that already matches Apple’s icon grid. A hard white square to the edges looks oversized in the Dock.
- Export a true 1024×1024 master; do not upscale a small logo.

After replacing `build/icon.png`, run `pnpm sign:dev` (or just `pnpm dev`) so the script rebuilds `build/icon.icns` and installs it into the local `Dagr.app` on macOS, or patches `electron.exe` with `build/icon.ico` on Windows.

| Generated / related file | Use |
| --- | --- |
| `build/icon.icns` | macOS bundle / About / Dock (built from `icon.png`) |
| `build/icon.ico` | Windows taskbar / packaging (built from `icon.png`) |
| `src/assets/app-icon.png` | In-app mark (`DagrMark` on auth, loading) |
| `public/app-icon.png` | Electron splash / static copy |

For the UI mark, copy the same artwork (or a simplified version) into `src/assets/app-icon.png` and `public/app-icon.png`.

Log in and sign up offer **Cloud** or **Self-hosted**. Cloud uses `VITE_DAGR_CLOUD_URL` (default `https://api.dagr.no`). Self-hosted asks for an API base URL (default from `VITE_DAGR_SELF_HOSTED_URL`, or `http://localhost:8080`). `VITE_DAGR_DEFAULT_MODE` picks the tab when nothing is stored (`selfhosted` for Compose and Coolify web builds). Keep the self-hosted URL in sync with `HTTP_ADDR` in `deploy/.env`. Run Postgres migrations and the API first (`make compose-up`, or `make migrate-up` then `make run`). The session token is stored in `sessionStorage`.

After signup the workspace rail is empty until you create a workspace (Add workspace). Creating one seeds `#general`. Channels, workspace invites, messages, scheduled sends, and notifications use the REST API (messages and notifications poll while signed in). Message URLs get rich link previews once the API worker unfurls Open Graph metadata (`make worker-run`).
