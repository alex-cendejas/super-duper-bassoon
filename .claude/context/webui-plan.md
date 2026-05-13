# Web UI Implementation Plan

## Executive Summary

Build a modern, Canonical-style Web UI for the super-duper-bassoon automation engine. The UI will provide full control and visibility into workflows, runs, clients, health metrics, alerts, and ban management. It will be packaged as a single-page application (SPA) served from the existing Go server binary using embedded assets.

---

## 1. Architecture & Technology Stack

### Core Technologies
- **Framework:** Vanilla framework (Canonical's CSS framework)
- **Language:** TypeScript (strict mode)
- **Module System:** ES modules (native)
- **Bundler:** Vite (for development and production builds)
- **State Management:** Custom store pattern (lightweight, no external dependencies)
- **API Communication:** Fetch API with custom service classes
- **Styling:** Vanilla CSS framework + custom SCSS/CSS as needed

### Key Design Principles
- **Server-Driven Updates:** UI reads from REST API; no WebSocket subscriptions initially (can add later)
- **Embedded Assets:** All static assets (HTML, JS, CSS) embedded in Go binary via `embed` package
- **Single HTML Entry Point:** One `index.html` served for all routes (SPA pattern)
- **TypeScript Strict:** Full type safety for all domain models and API contracts
- **Accessibility:** Follow WCAG 2.1 AA standards using Vanilla framework's built-in patterns

---

## 2. Project Structure

```
/project/webui/
├── .claude/
│   └── context/
│       └── webui-plan.md          (this file)
├── src/
│   ├── index.html                 (entry point, minimal markup)
│   ├── main.ts                    (app initialization)
│   ├── app.ts                     (root component/app controller)
│   ├── router.ts                  (client-side routing)
│   ├── styles/
│   │   ├── main.scss              (global styles + Vanilla overrides)
│   │   ├── variables.scss         (color, spacing, typography vars)
│   │   ├── components.scss        (reusable component styles)
│   │   └── pages.scss             (page-specific styles)
│   ├── api/
│   │   ├── client.ts              (HTTP client wrapper)
│   │   ├── workflows.ts           (workflow API service)
│   │   ├── runs.ts                (run API service)
│   │   ├── clients.ts             (client API service)
│   │   ├── health.ts              (health metrics API service)
│   │   ├── bans.ts                (ban management API service)
│   │   ├── alerts.ts              (alert API service)
│   │   └── system.ts              (system status API service)
│   ├── types/
│   │   ├── index.ts               (all domain types exported)
│   │   ├── workflow.ts            (Workflow, TriggerSpec, etc.)
│   │   ├── run.ts                 (Run, RunHealth, etc.)
│   │   ├── client.ts              (ClientMetadata, etc.)
│   │   ├── ban.ts                 (BanRecord, etc.)
│   │   ├── health.ts              (RunHealth, TypeHealth, etc.)
│   │   ├── alert.ts               (Alert types)
│   │   ├── circuit.ts             (CircuitBreaker state)
│   │   └── api.ts                 (request/response envelope types)
│   ├── store/
│   │   ├── index.ts               (store initialization)
│   │   ├── workflows.ts           (workflow state slice)
│   │   ├── runs.ts                (run state slice)
│   │   ├── clients.ts             (client state slice)
│   │   ├── health.ts              (health metrics state slice)
│   │   ├── bans.ts                (ban state slice)
│   │   ├── alerts.ts              (alert state slice)
│   │   └── ui.ts                  (UI state: sidebar open, current page, etc.)
│   ├── components/
│   │   ├── layout/
│   │   │   ├── Header.ts          (top navigation bar)
│   │   │   ├── Sidebar.ts         (left navigation menu)
│   │   │   └── MainLayout.ts      (layout wrapper)
│   │   ├── common/
│   │   │   ├── LoadingSpinner.ts
│   │   │   ├── ErrorAlert.ts
│   │   │   ├── ConfirmDialog.ts
│   │   │   ├── Badge.ts
│   │   │   ├── HealthBar.ts       (visual health gauge)
│   │   │   └── Pagination.ts
│   │   ├── workflows/
│   │   │   ├── WorkflowsList.ts   (table view of all workflows)
│   │   │   ├── WorkflowCard.ts    (summary card)
│   │   │   ├── WorkflowDetail.ts  (detail + edit modal)
│   │   │   ├── WorkflowForm.ts    (create/edit form)
│   │   │   └── WorkflowActions.ts (trigger, activate, deactivate buttons)
│   │   ├── runs/
│   │   │   ├── RunsList.ts        (paginated table of runs)
│   │   │   ├── RunCard.ts         (summary card)
│   │   │   ├── RunDetail.ts       (detail page with results grid)
│   │   │   ├── ResultsGrid.ts     (results per client in run)
│   │   │   └── RunTimeline.ts     (visual timeline of run state changes)
│   │   ├── clients/
│   │   │   ├── ClientsList.ts     (table of all registered clients)
│   │   │   ├── ClientDetail.ts    (detail page with state inspection)
│   │   │   ├── ClientBadges.ts    (display OS, active status, labels)
│   │   │   ├── LabelEditor.ts     (edit client labels)
│   │   │   ├── StateInspector.ts  (view inner_state JSON with syntax highlight)
│   │   │   └── ClientFilter.ts    (filter/search controls)
│   │   ├── health/
│   │   │   ├── HealthDashboard.ts (overview of all workflow health)
│   │   │   ├── HealthCard.ts      (per-workflow health summary)
│   │   │   ├── TrendChart.ts      (sparkline trend visualization)
│   │   │   └── RunHealthDetail.ts (detailed per-run health metrics)
│   │   ├── bans/
│   │   │   ├── BansList.ts        (table of active bans)
│   │   │   ├── BanDetail.ts       (detail with evidence & unban action)
│   │   │   └── UnbanForm.ts       (unban confirmation dialog)
│   │   └── alerts/
│   │       ├── AlertsList.ts      (chronological alert log)
│   │       ├── AlertDetail.ts     (detail with context)
│   │       ├── AlertSeverity.ts   (display severity badge)
│   │       └── AlertFilters.ts    (filter by severity, type, time range)
│   ├── pages/
│   │   ├── WorkflowsPage.ts       (workflows menu container)
│   │   ├── RunsPage.ts            (runs menu container)
│   │   ├── HealthPage.ts          (health dashboard container)
│   │   ├── ClientsPage.ts         (clients menu container)
│   │   ├── AlertsPage.ts          (alerts container)
│   │   ├── BansPage.ts            (bans management container)
│   │   └── NotFoundPage.ts        (404 fallback)
│   ├── utils/
│   │   ├── format.ts              (date, duration, percentage formatting)
│   │   ├── color.ts               (health-based color mapping)
│   │   ├── validation.ts          (form field validation)
│   │   ├── http.ts                (HTTP status code handling)
│   │   └── dom.ts                 (DOM manipulation helpers)
│   └── config.ts                  (API base URL, etc.)
├── public/
│   └── favicon.ico
├── package.json
├── tsconfig.json
├── vite.config.ts
├── .eslintrc.json
├── .gitignore
└── README.md
```

---

## 3. Integration with Go Server Binary

### Embedded Assets Strategy

**Location:** The webui is built to `/webui/dist/` (via Vite), then embedded in the Go binary.

**Implementation Steps:**

1. **Add `//go:embed` directive** in `/project/cmd/main.go`:
   ```go
   //go:embed webui/dist
   var webuiAssets embed.FS
   ```

2. **Register Handler in HTTP Router** (`/project/internal/adapters/http/server.go`):
   ```go
   // Serve webui SPA
   r.Handle("/*", http.FileServer(http.FS(webuiAssets)))
   
   // API routes registered first, so /api/* is not caught by SPA fallback
   r.Post("/api/workflows", ...)
   ```

3. **SPA Fallback:** Configure Vite to generate a single `index.html` at the root of `dist/`. The Go handler serves this for any route not matching an API endpoint, enabling client-side routing.

4. **Build Pipeline:**
   - `cd webui && npm run build` → produces `dist/` folder
   - `go build ./cmd` → embeds `dist/` into binary
   - Single executable ships with both server and UI

### API Base URL

- Development: `http://localhost:8080` (configured in `vite.config.ts`)
- Production: Relative paths to same origin (e.g., `/api/workflows`)

---

## 4. Navigation Structure

### Top-Level Menu (Sidebar)

Located in `Sidebar.ts`, persistent across all pages. Items:

1. **Workflows**
   - Create Workflow (button)
   - List Workflows (main view)
   - Link: `/workflows`

2. **Runs**
   - List Runs (paginated, with filters)
   - Link: `/runs`

3. **Health**
   - Health Dashboard (overview of all workflow_types)
   - Link: `/health`

4. **Clients**
   - List Clients (with state, labels, active status)
   - Link: `/clients`

5. **Alerts**
   - Alert Log (chronological, filterable by severity/type)
   - Link: `/alerts`

6. **Bans**
   - Active Bans (with evidence, unban actions)
   - Link: `/bans`

### Header

- Logo + branding (Canonical style)
- System status indicator (NATS connected, DB healthy, uptime)
- User menu placeholder (for future RBAC)
- Refresh/polling controls

---

## 5. Page-by-Page Breakdown

### 5.1 Workflows Page (`/workflows`)

**Purpose:** Create, list, edit, trigger, and manage workflow definitions.

**Components:**
- **WorkflowsList:** Table view with columns:
  - Name, Workflow Type, Activity, Status (Active/Inactive), Created, Actions
  - Inline actions: View, Edit, Trigger, Activate/Deactivate, Delete
  - Bulk select for batch actions (future)

- **WorkflowDetail Modal:** Opens on "View" click
  - Display all fields (read-only view of current state)
  - Button to open edit form
  - Recent runs section (last 5 runs of this workflow)

- **WorkflowForm Modal:** Create/Edit
  - Input fields: Name, Description, WorkflowType
  - Activity selector (dropdown with ActivityType options)
  - Params editor (JSON schema form builder, or simple key-value editor)
  - Target Filter input (with syntax hint/validator)
  - Trigger section:
    - Kind selector (scheduled, event, state_change, manual)
    - Conditional inputs (cron for scheduled, on_complete for event, etc.)
  - Thresholds: Success Threshold (0-100%), Loop Threshold (ms), Timeout (ms)
  - Enable/Disable toggle
  - Submit/Cancel buttons

- **WorkflowActions:**
  - Trigger button → opens TriggerDialog (optional reason field)
  - Activate button (if inactive)
  - Deactivate button (if active) → shows deactivation reason

**Data Requirements:**
- GET `/workflows` → list all workflows
- POST `/workflows` → create new workflow
- GET `/workflows/{id}` → get workflow details
- PUT `/workflows/{id}` → update workflow
- DELETE `/workflows/{id}` → delete workflow
- POST `/workflows/{id}/trigger` → trigger manual run
- POST `/workflows/{id}/activate` → reactivate workflow
- POST `/workflows/{id}/deactivate` → deactivate workflow (circuit breaker override)

---

### 5.2 Runs Page (`/runs`)

**Purpose:** Monitor workflow runs and their results across clients.

**Components:**
- **RunsList:** Table view with columns:
  - Run ID, Workflow, Triggered, State (pending/in_progress/completed/failed), Success %, Clients (N/total), Actions
  - Inline action: View Details
  - Filters: Workflow Type, Date Range, State
  - Pagination (10, 25, 50 per page)

- **RunDetail Page:** Detailed view of a single run
  - Header: Run ID, Workflow Type, Triggered At, State, Duration
  - Health Summary Card: Total Clients, Success %, Fail %, Error %, Pending %
  - ResultsGrid: Table of client results
    - Client ID, OS, Labels, Activity, Status (success/fail/error), Timestamp, Payload/Error
    - Sortable, filterable
  - RunTimeline (optional): Visual representation of run lifecycle

**Data Requirements:**
- GET `/runs` → list runs (paginated, filterable)
- GET `/runs/{id}` → get run details with health snapshot
- GET `/runs/{id}/results` → get all results for a run
- GET `/workflows/{id}/runs` → get runs for a specific workflow

---

### 5.3 Health Page (`/health`)

**Purpose:** Dashboard for monitoring workflow health and circuit breaker status.

**Components:**
- **HealthDashboard:** Overview grid/cards
  - One card per workflow_type
  - Each card shows:
    - Workflow Type name
    - Current health: Success %, Fail %, Error %
    - Trend (improving/degrading/stable) with sparkline
    - Status badge: Green (above threshold), Red (below threshold, circuit broken)
    - Link to detailed health view

- **HealthCard:** Individual workflow health summary
  - Metrics: Total runs (last N), avg success %, fail %, error %
  - Threshold indicator (success_threshold vs. current)
  - Circuit breaker status (active/deactivated + reason)
  - Last run timestamp

- **RunHealthDetail:** Detailed per-run health breakdown
  - Comprehensive metrics table
  - Client participation: Total, Success, Fail, Error, Pending, Banned
  - Percentages calculated excluding banned clients
  - Visual indicators (green/yellow/red bars)

- **TrendChart:** Sparkline showing health trend across last N runs (future: upgrade to chart library)

**Data Requirements:**
- GET `/health` → all workflow health metrics
- GET `/health/{workflow_type}` → health aggregates for specific workflow_type
- GET `/circuits` → circuit breaker state for all workflows
- GET `/circuits/{workflow_id}` → circuit breaker state for specific workflow

---

### 5.4 Clients Page (`/clients`)

**Purpose:** Register, view, and manage client metadata.

**Components:**
- **ClientsList:** Table view with columns:
  - Client ID, OS, Status (Active/Inactive), Labels, Last Seen, State Snapshot, Actions
  - Inline actions: View, Edit Labels
  - Filters: OS, Active Status, Label search
  - Pagination

- **ClientDetail Page:** Full client inspection
  - Header: Client ID, OS, Last Seen, Active Status
  - Badge Section: OS badge, Active status badge, Label pills (editable)
  - StateInspector: JSON display of inner_state (syntax-highlighted, expandable)
  - Client Bans Section: Link to bans affecting this client (if any)
  - Associated Runs: Recent runs targeting this client

- **LabelEditor Modal:** Add/remove/edit client labels
  - Key-value pair input
  - Submit button (PUT `/clients/{id}/labels`)

- **ClientFilter Controls:**
  - OS dropdown (filter to selected OS)
  - Active Status toggle
  - Search input (client_id, labels, state fields)

**Data Requirements:**
- GET `/clients` → list all registered clients
- GET `/clients/{id}` → client metadata with inner_state
- PUT `/clients/{id}` → update client metadata (labels, active status)
- (Optional: DELETE `/clients/{id}` → deregister client)

---

### 5.5 Alerts Page (`/alerts`)

**Purpose:** View alerts triggered by the automation engine.

**Components:**
- **AlertsList:** Chronological log of alerts
  - Columns: Timestamp, Severity (info/warning/error/critical), Type, Message, Source Workflow, Actions
  - Reverse chronological (newest first)
  - Filters: Severity, Type, Workflow, Date Range
  - Pagination (25 per page)
  - Inline action: View Details

- **AlertDetail Page:** Full alert context
  - Header: Alert ID, Timestamp, Severity badge, Type, Source Workflow
  - Message (full text)
  - Metadata: Client affected (if applicable), Run ID evidence, Details JSON
  - Related Entities: Links to related workflow, run, client, ban (if applicable)

- **AlertSeverity Badge:** Color-coded severity indicator
  - info → blue
  - warning → yellow
  - error → orange
  - critical → red

- **AlertFilters:** Sidebar filter controls
  - Severity checkboxes
  - Type dropdown
  - Workflow Type dropdown
  - Date range picker

**Data Requirements:**
- GET `/alerts` → list alerts (paginated, filterable)
- GET `/alerts/{id}` → alert detail
- (Optional: POST `/alerts/{id}/acknowledge` → mark as read)

---

### 5.6 Bans Page (`/bans`)

**Purpose:** Manage banned clients and review ban evidence.

**Components:**
- **BansList:** Table of active bans
  - Columns: Client ID, Workflow Type, Ban Reason (loop_detected/manual/admin_unban_failed), Banned At, Banned Until (if temporary), Status (Active/Expired), Actions
  - Filters: Client ID, Workflow Type, Reason, Status
  - Pagination
  - Inline action: View Details, Unban (if active)

- **BanDetail Page:** Full ban review
  - Header: Client ID, Workflow Type, Status (Active/Expired)
  - Ban Metadata:
    - Reason, Banned At, Banned Until, Banned By, Active flag
  - Evidence Section:
    - Run ID Evidence (link to run)
    - Result Evidence (JSON display of the result that triggered ban)
  - UnbanForm button (if active)

- **UnbanForm Modal:** Confirm unban action
  - Admin ID input (required)
  - Reason input (required)
  - Confirmation checkbox
  - Submit/Cancel buttons
  - API call: PUT `/bans/{client_id}/unban` with UnbanRequest body

**Data Requirements:**
- GET `/bans` → list all ban records
- GET `/bans/{client_id}` → bans for a specific client
- PUT `/bans/{client_id}/unban` → unban client (for specific workflow_type)

---

## 6. Shared Components & Patterns

### Common Components

- **LoadingSpinner:** Animated spinner with optional message
- **ErrorAlert:** Error toast/alert with dismissal
- **ConfirmDialog:** Reusable confirmation modal (for destructive actions)
- **Badge:** Status badge (Active, Inactive, Success, Failed, etc.)
- **HealthBar:** Horizontal bar chart showing success/fail/error/pending percentages
- **Pagination:** Reusable pagination control (prev, page numbers, next, per-page dropdown)

### Layout

- **MainLayout:** Wrapper component combining Header + Sidebar + PageContent
- **Header:** Fixed top bar with logo, system status, user menu
- **Sidebar:** Fixed left navigation with collapsible menu

### Styling Approach

- **Vanilla Framework:** Use official Vanilla CSS framework classes
- **Custom SCSS:** Override and extend for Canonical brand colors
- **CSS Variables:** Define color palette, spacing scale, typography in `variables.scss`
- **BEM Methodology:** For custom component classes (e.g., `.health-card__metrics`)
- **Responsive Design:** Mobile-first, breakpoints via Vanilla framework

---

## 7. Store/State Management

### Store Pattern

A simple, custom store pattern (no Redux/Zustand):

```typescript
// store/index.ts
interface AppState {
  workflows: {
    items: Workflow[],
    loading: boolean,
    error?: string,
  },
  runs: {
    items: Run[],
    total: number,
    page: number,
    limit: number,
    loading: boolean,
  },
  // ... other slices
  ui: {
    sidebarOpen: boolean,
    currentPage: string,
    modals: { [key: string]: boolean },
  },
}

// Publish-subscribe pattern
class Store {
  state: AppState;
  listeners: Set<() => void>;
  
  subscribe(listener: () => void) { ... }
  setState(slice: keyof AppState, updates: Partial<AppState[slice]>) { ... }
}
```

### Data Flow

1. **User interaction** → action (e.g., click "Create Workflow")
2. **Component calls API service** (e.g., `workflowsAPI.create(...)`)
3. **API service** updates store state
4. **Store notifies all listeners**
5. **Components re-render** based on new state

### Caching & Polling

- **Simple TTL Cache:** Keep fetched data in store, refresh on demand or at intervals
- **Polling:** Optional auto-refresh (e.g., refresh runs list every 5 seconds)
- **No Real-Time:** Initial implementation uses polling/manual refresh; WebSocket upgrade is future work

---

## 8. API Service Layer

### Client Wrapper (`api/client.ts`)

```typescript
export class APIClient {
  baseURL: string;
  
  async request(
    method: 'GET' | 'POST' | 'PUT' | 'DELETE',
    path: string,
    body?: unknown
  ): Promise<unknown> {
    const response = await fetch(`${this.baseURL}${path}`, {
      method,
      headers: { 'Content-Type': 'application/json' },
      body: body ? JSON.stringify(body) : undefined,
    });
    
    if (!response.ok) throw new APIError(...);
    return response.json();
  }
}
```

### Service Modules

- **workflows.ts:** `getAll()`, `get(id)`, `create(req)`, `update(id, req)`, `delete(id)`, `trigger(id, reason)`, `activate(id)`, `deactivate(id)`
- **runs.ts:** `getAll(filters)`, `get(id)`, `getResults(id)`
- **clients.ts:** `getAll(filters)`, `get(id)`, `updateLabels(id, labels)`
- **health.ts:** `getAll()`, `get(workflowType)`
- **bans.ts:** `getAll(filters)`, `get(clientId)`, `unban(clientId, workflowType, req)`
- **alerts.ts:** `getAll(filters)`, `get(id)`
- **system.ts:** `getStatus()` (for health checks in header)

### Error Handling

```typescript
class APIError extends Error {
  constructor(public code: string, public status: number, message: string) {
    super(message);
  }
}
```

Map API error codes to user-friendly messages in error handling middleware.

---

## 9. Type Safety

### Type Definitions

All types mirrored from Go domain models, located in `src/types/`:

```typescript
// types/workflow.ts
export interface Workflow {
  id: string;
  name: string;
  description: string;
  workflow_type: string;
  activity: ActivityType;
  params: Record<string, unknown>;
  target_filter: string;
  trigger: TriggerSpec;
  success_threshold: number;
  loop_threshold_ms: number;
  timeout_ms: number;
  active: boolean;
  created_at: string; // ISO 8601
  updated_at: string;
  deactivated_reason?: string;
}

export type ActivityType =
  | 'reboot'
  | 'install_package'
  | 'upgrade_package'
  | 'remove_package'
  | 'apply_config'
  | 'validate_config'
  | 'run_script';
```

### Request/Response Types

```typescript
// types/api.ts
export interface CreateWorkflowRequest { ... }
export interface EditWorkflowRequest { ... }
export interface TriggerWorkflowRequest { ... }
export interface UnbanRequest { ... }
```

---

## 10. Build & Development Setup

### `package.json` Dependencies

- `vite` - bundler
- `typescript` - language
- `vanilla-framework` - CSS framework
- `sass` - preprocessor (optional, for nested styles)
- `eslint`, `prettier` - code quality

### `vite.config.ts`

```typescript
export default {
  root: 'src',
  build: {
    outDir: '../dist',
    emptyOutDir: true,
    target: 'es2020',
  },
  server: {
    proxy: {
      '/api': 'http://localhost:8080',
    },
  },
};
```

### `tsconfig.json`

- `strict: true`
- `target: es2020`
- `module: esnext`
- `moduleResolution: bundler`

### `package.json` Scripts

```json
{
  "scripts": {
    "dev": "vite",
    "build": "tsc && vite build",
    "preview": "vite preview",
    "lint": "eslint src --ext .ts,.tsx",
    "format": "prettier --write src"
  }
}
```

### Embedding in Go Binary

**In `cmd/main.go`:**

```go
import "embed"

//go:embed webui/dist
var webuiAssets embed.FS

func main() {
  // ... existing setup ...
  
  // Register SPA handler last so /api routes take precedence
  r.Handle("/*", http.FileServer(http.FS(webuiAssets)))
}
```

### Build Artifacts

After `npm run build` in `/project/webui`:
- `/project/webui/dist/index.html` - entry point
- `/project/webui/dist/assets/` - bundled JS, CSS
- `/project/webui/dist/assets/...` - images, fonts (if any)

All files are embedded at build time.

---

## 11. Canonical Design Compliance

### Visual Guidelines
- **Color Palette:** Use Canonical brand colors via CSS variables
  - Primary: Ubuntu Orange (#E95420)
  - Secondary: Warm Neutral (#FFF7F5)
  - Accent: Canonical Blue (#0068D6)
  - Text: Dark Gray/Black (#111111)
  - Borders: Light Gray (#CDCDD4)

- **Typography:** Prefer sans-serif (system font stack with Ubuntu font as fallback)
- **Spacing:** Use 8px grid (8, 16, 24, 32, 48, 64px)
- **Component Patterns:** Leverage Vanilla framework defaults (buttons, forms, cards, tables)

### Canonical UI Patterns
- **Sidebar Navigation:** Persistent left menu (Vanilla `p-navigation` component)
- **Cards:** Used for workflow/client summaries with `p-card` class
- **Tables:** Use Vanilla `p-table` for data grids
- **Modals:** Use Vanilla `p-modal` component for forms/dialogs
- **Alerts/Toasts:** Use Vanilla alert components for errors/success messages
- **Badges:** Use Vanilla badge classes for status indicators

### Accessibility
- Semantic HTML5 elements
- ARIA labels for interactive elements
- Keyboard navigation support (Tab, Enter, Escape)
- Color contrast >= 4.5:1 for text
- Focus indicators on all interactive elements
- Test with WCAG 2.1 Level AA checklist

---

## 12. Development Phases

### Phase 1: Foundation (Week 1)
- [ ] Set up Vite + TypeScript + Vanilla framework
- [ ] Scaffold project structure
- [ ] Implement routing (client-side)
- [ ] Create layout (Header, Sidebar, MainLayout)
- [ ] Implement store pattern
- [ ] Create common components (LoadingSpinner, ErrorAlert, Badge, etc.)
- [ ] Implement API client wrapper
- [ ] Define all TypeScript types

### Phase 2: Core Pages (Week 2-3)
- [ ] WorkflowsPage + components (list, create, edit, trigger, delete)
- [ ] RunsPage + components (list, detail, results grid)
- [ ] ClientsPage + components (list, detail, labels)
- [ ] HealthPage + components (dashboard, metrics, trends)
- [ ] BansPage + components (list, detail, unban)
- [ ] AlertsPage + components (list, detail, filters)

### Phase 3: Polish & Integration (Week 4)
- [ ] Vanilla framework styling integration
- [ ] Canonical brand colors & design refinements
- [ ] Form validation & error handling
- [ ] API error handling & user feedback
- [ ] Loading states & skeleton screens
- [ ] Pagination & sorting
- [ ] Auto-refresh/polling for key pages
- [ ] Accessibility audit (keyboard nav, ARIA, color contrast)
- [ ] ESLint & TypeScript strict mode pass
- [ ] Go embedding & binary build integration
- [ ] E2E testing (optional)

### Phase 4: Advanced Features (Future)
- [ ] WebSocket support for real-time updates
- [ ] Advanced charting (health trends, run timelines)
- [ ] Bulk actions (batch trigger, ban management)
- [ ] User authentication & RBAC
- [ ] Workflow templates
- [ ] Export/import workflows
- [ ] Dark mode toggle

---

## 13. Key Implementation Details

### Client-Side Routing

Use a simple hash-based router (`#/workflows`, `#/runs/{id}`, etc.):

```typescript
// router.ts
class Router {
  private route = writable<string>('#');
  
  constructor() {
    window.addEventListener('hashchange', () => {
      this.route.set(window.location.hash);
    });
  }
  
  navigate(path: string) {
    window.location.hash = path;
  }
  
  subscribe(fn: (route: string) => void) {
    return this.route.subscribe(fn);
  }
}
```

### Modal/Dialog Management

Store modal open state in store, trigger via actions:

```typescript
// actions
store.openModal('createWorkflow');
store.closeModal('createWorkflow');

// component
if (store.state.ui.modals['createWorkflow']) {
  <WorkflowForm onSubmit={onSubmit} onCancel={onCancel} />
}
```

### Form Validation

Simple validation utilities:

```typescript
// utils/validation.ts
export function validateWorkflowName(name: string): string | null {
  if (!name || name.trim().length === 0) {
    return 'Name is required';
  }
  if (name.length > 255) {
    return 'Name must be <= 255 characters';
  }
  return null;
}
```

### Polling & Auto-Refresh

Optional polling for key pages (runs, health):

```typescript
// pages/RunsPage.ts
function refreshRuns() {
  runsAPI.getAll(...).then(runs => {
    store.setState('runs', { items: runs });
  });
}

setInterval(refreshRuns, 5000); // Refresh every 5s
```

---

## 14. Error Handling & User Feedback

### API Error Response Format

Expected from Go API:

```json
{
  "code": "VALIDATION_ERROR",
  "message": "invalid JSON",
  "details": { /* optional */ }
}
```

### Error Handling in Components

```typescript
try {
  const workflow = await workflowsAPI.create(req);
  store.setState('workflows', { items: [...items, workflow] });
  showSuccessToast('Workflow created');
} catch (err) {
  if (err instanceof APIError) {
    showErrorToast(`${err.code}: ${err.message}`);
  } else {
    showErrorToast('An unexpected error occurred');
  }
}
```

### User Feedback Patterns

- **Success Toast:** "Workflow created", auto-dismiss after 3s
- **Error Toast:** Full error message, dismissible, persist until dismissed
- **Loading State:** Spinner overlay or skeleton on component
- **Validation Errors:** Inline field error messages, red border on input
- **Confirmation Dialog:** For destructive actions (delete, unban)

---

## 15. Performance Considerations

### Initial Load
- Lazy load pages (only load components on route change)
- Inline critical CSS in `index.html`
- Minify JS/CSS in production build

### Data Fetching
- Cache API responses in store
- Implement request deduplication (avoid duplicate API calls in flight)
- Use pagination for large lists (default 10, configurable)

### Rendering
- Component memoization (if using re-render frameworks in future)
- Virtual scrolling for large tables (future optimization)
- Debounce user input (search, filter)

---

## 16. Testing Strategy (Optional for MVP)

### Unit Tests
- TypeScript type checking (strict mode catches most errors)
- Utility functions (format, validation, color)

### Integration Tests
- API service layer (mock fetch responses)
- Store state updates

### E2E Tests (Future)
- Critical user flows (create workflow, trigger run, unban client)
- Use Playwright or Cypress

---

## 17. Deployment & Operations

### Single Binary Deployment
- Build UI: `cd webui && npm run build`
- Build binary: `cd project && go build -o super-duper-bassoon ./cmd`
- Deploy: Single `super-duper-bassoon` executable with embedded UI

### Configuration
- UI accesses API at same origin (`/api/...`)
- API base URL configurable via `config.ts` (for dev vs. prod)

### Monitoring
- Browser console for client-side errors
- Network tab for API request debugging
- Server logs for backend issues

---

## 18. Future Enhancements

1. **Real-Time Updates:** Upgrade from polling to WebSocket/Server-Sent Events
2. **Advanced Analytics:** Charts for health trends, workflow duration trends
3. **Bulk Operations:** Batch trigger, batch ban/unban
4. **Workflow Templates:** Pre-built workflow definitions
5. **Audit Trail:** Log all admin actions (create, edit, trigger, unban)
6. **RBAC:** Role-based access control (viewer, operator, admin)
7. **Dark Mode:** Theme toggle
8. **Export/Import:** Backup and restore workflows
9. **Notifications:** Browser notifications for critical alerts
10. **Mobile Responsive:** Full mobile UI (currently desktop-focused)

---

## 19. Success Criteria

- [ ] All 6 menu sections functional and visually consistent
- [ ] Full CRUD operations on workflows, clients, bans
- [ ] Health dashboard displays accurate metrics
- [ ] Alerts and bans management operational
- [ ] Vanilla framework styling applied consistently
- [ ] TypeScript strict mode passes
- [ ] API integration complete and tested
- [ ] Embedded in Go binary and deployable as single executable
- [ ] Keyboard navigation and accessibility compliant
- [ ] Performance: initial load < 2s, API responses < 1s
- [ ] All routes accessible and no 404s (except intentional)
- [ ] Error handling and user feedback clear

---

## 20. File Checklist

### To Create
- [ ] `/project/webui/src/` (all subdirectories and files per structure above)
- [ ] `/project/webui/package.json`
- [ ] `/project/webui/tsconfig.json`
- [ ] `/project/webui/vite.config.ts`
- [ ] `/project/webui/.eslintrc.json`
- [ ] `/project/webui/README.md`

### To Modify
- [ ] `/project/cmd/main.go` - add `//go:embed webui/dist` and register SPA handler
- [ ] `/project/internal/adapters/http/server.go` - register SPA fallback route
- [ ] `/project/go.mod` - ensure no new dependencies needed (Go side)
- [ ] `/project/.gitignore` - add `/webui/node_modules`, `/webui/dist`

---

## Questions for Clarification

1. **Authentication:** Should the UI have login? (Assumption: No auth for MVP)
2. **Polling Frequency:** Preferred polling interval for runs and health updates? (Suggestion: 5-10s)
3. **Data Persistence:** Should client-side store persist across page reloads? (Suggestion: No, fetch fresh)
4. **Bulk Operations:** Required for MVP or future enhancement? (Assumption: Future)
5. **Chart Library:** For health trend sparklines, use Vanilla-compatible library? (Suggestion: Keep simple SVG for MVP)
6. **API Versioning:** Should the WebUI version-lock to specific API version? (Assumption: No, always latest)
7. **Logging:** Client-side error logging to server, or console only? (Suggestion: Console for MVP)

---

This plan provides a complete blueprint for implementing a production-quality Web UI for the super-duper-bassoon automation engine, fully integrated with the Go server binary and styled according to Canonical design standards.
