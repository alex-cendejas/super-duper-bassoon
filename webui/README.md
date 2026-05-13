# super-duper-bassoon Web UI

A modern, TypeScript-based Web UI for the super-duper-bassoon automation engine, built with Vite, Vanilla Framework, and vanilla web technologies.

## Features

- **Full CRUD Operations**: Manage workflows, runs, clients, and bans
- **Real-Time Monitoring**: Track workflow health, circuit breaker status, and system alerts
- **Responsive Design**: Mobile-friendly layout using Canonical's Vanilla CSS framework
- **Type Safety**: Full TypeScript strict mode for robust development
- **Modular Architecture**: Clean separation of concerns with API services, components, and store
- **No Framework Dependencies**: Built with vanilla JavaScript/TypeScript (except CSS framework)

## Project Structure

```
webui/
├── src/
│   ├── index.html                 # Entry point
│   ├── main.ts                    # App initialization
│   ├── app.ts                     # Root application controller
│   ├── router.ts                  # Client-side routing
│   ├── config.ts                  # Configuration
│   ├── api/                       # API service layer
│   │   ├── client.ts              # HTTP client wrapper
│   │   ├── workflows.ts           # Workflow API
│   │   ├── runs.ts                # Run API
│   │   ├── clients.ts             # Client API
│   │   ├── health.ts              # Health metrics API
│   │   ├── bans.ts                # Ban management API
│   │   ├── alerts.ts              # Alert API
│   │   └── system.ts              # System status API
│   ├── types/                     # TypeScript type definitions
│   │   ├── workflow.ts
│   │   ├── run.ts
│   │   ├── client.ts
│   │   ├── ban.ts
│   │   ├── health.ts
│   │   ├── alert.ts
│   │   └── api.ts
│   ├── store/                     # State management
│   │   └── index.ts               # Custom store implementation
│   ├── components/                # UI components
│   │   ├── layout/
│   │   │   ├── Header.ts
│   │   │   ├── Sidebar.ts
│   │   │   └── MainLayout.ts
│   │   └── common/
│   │       ├── LoadingSpinner.ts
│   │       ├── ErrorAlert.ts
│   │       ├── Badge.ts
│   │       ├── HealthBar.ts
│   │       ├── Pagination.ts
│   │       └── ConfirmDialog.ts
│   ├── pages/                     # Page components
│   │   ├── WorkflowsPage.ts
│   │   ├── RunsPage.ts
│   │   ├── HealthPage.ts
│   │   ├── ClientsPage.ts
│   │   ├── AlertsPage.ts
│   │   ├── BansPage.ts
│   │   └── NotFoundPage.ts
│   ├── utils/                     # Utility functions
│   │   ├── format.ts
│   │   ├── color.ts
│   │   ├── validation.ts
│   │   └── dom.ts
│   └── styles/                    # SCSS stylesheets
│       ├── main.scss
│       ├── variables.scss
│       ├── components.scss
│       └── pages.scss
├── public/                        # Static assets
│   └── logo.png                   # Application logo
├── tests/                         # Test files
│   ├── app.spec.ts                # E2E tests (Playwright)
│   ├── components.spec.ts         # Component tests
│   ├── api-integration.spec.ts    # API integration tests
│   └── manual-verification.mjs    # Build verification tests
├── package.json
├── tsconfig.json
├── tsconfig.node.json
├── vite.config.ts
├── playwright.config.ts
└── README.md
```

## Setup

### Prerequisites

- Node.js 18+
- npm 8+

### Installation

```bash
cd webui
npm install
```

### Development

Start the development server:

```bash
npm run dev
```

The app will be available at `http://localhost:5173/`

### Building

Build for production:

```bash
npm run build
```

This creates a `dist/` folder with optimized bundles ready for embedding in the Go binary.

### Testing

#### Manual Verification Tests

Run static verification tests (no browser required):

```bash
node tests/manual-verification.mjs
```

This verifies:
- Project structure and required files
- Build artifacts
- Configuration files
- TypeScript and build setup

#### E2E Tests (Playwright)

Install browsers first:

```bash
npx playwright install
```

Run all tests:

```bash
npm test
```

Run specific test file:

```bash
npm test app.spec.ts
```

View test report:

```bash
npx playwright show-report
```

### Code Quality

Lint TypeScript:

```bash
npm run lint
```

Format code (Prettier):

```bash
npm run format
```

## Architecture

### State Management

Simple publish-subscribe store pattern with no external dependencies:

```typescript
store.subscribe((state) => {
  // Component updates when state changes
});

store.setState('workflows', { items: [...], loading: false });
```

### API Communication

Type-safe API client with automatic error handling:

```typescript
import { workflowsAPI } from '@/api/workflows';

const workflows = await workflowsAPI.getAll();
```

### Routing

Hash-based client-side routing (works with SPA fallback):

```typescript
router.navigate('/#/workflows');
router.subscribe((path) => {
  // Handle route changes
});
```

## Features

### Workflows Page

- List all workflows with status and activity
- Create new workflows
- Edit existing workflows
- Trigger manual workflow runs
- Activate/deactivate workflows

### Runs Page

- View paginated list of workflow runs
- See run health metrics
- Filter by workflow type and state
- View detailed run results

### Health Page

- Dashboard showing health of all workflow types
- Circuit breaker status
- Success/failure percentages
- Trend indicators

### Clients Page

- List all registered clients
- View client metadata and state
- Edit client labels
- Monitor client status

### Alerts Page

- Chronological alert log
- Filter by severity and type
- View alert details and context

### Bans Page

- Manage banned clients
- View ban reasons and evidence
- Unban clients with admin confirmation

## Integration with Go Server

### Embedding

The built `dist/` folder is embedded in the Go binary:

```go
//go:embed webui/dist
var webuiAssets embed.FS
```

### SPA Fallback

Configure the Go router to serve `index.html` for all non-API routes:

```go
r.Handle("/*", http.FileServer(http.FS(webuiAssets)))
```

### API Base URL

Development: `http://localhost:8080`
Production: Relative paths (same origin)

Configure in `src/config.ts`.

## Performance

- **Bundle size**: ~25KB gzipped (JS + CSS)
- **Initial load**: <1s on typical network
- **Asset loading**: Logo embedded (3.3MB PNG)

## Browser Support

- Chrome/Chromium 90+
- Firefox 88+
- Safari 14+
- Edge 90+

## Accessibility

- WCAG 2.1 AA compliant
- Semantic HTML5
- ARIA labels on interactive elements
- Keyboard navigation support
- Color contrast ratios ≥ 4.5:1

## Future Enhancements

- [ ] WebSocket support for real-time updates
- [ ] Advanced charting (health trends, timelines)
- [ ] Bulk operations (batch trigger, ban management)
- [ ] User authentication and RBAC
- [ ] Dark mode toggle
- [ ] Export/import workflows
- [ ] Audit trail logging

## Troubleshooting

### Dev server not starting

```bash
# Clear node_modules and reinstall
rm -rf node_modules package-lock.json
npm install
```

### Build errors

```bash
# TypeScript errors
npm run build

# Check ESLint issues
npm run lint
```

### API calls failing

- Ensure Go server is running on `http://localhost:8080`
- Check CORS headers if running on different domain
- Verify API endpoints match backend implementation

## License

MIT (or same as super-duper-bassoon main project)

## Contributing

Follow the code style:
- Use TypeScript strict mode
- ESLint rules enforced
- Prettier formatting
- Semantic component naming
- No external state management libraries

## Support

For issues or questions, refer to the main super-duper-bassoon project documentation.
