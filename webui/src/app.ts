import { router } from '@/router';
import { store } from '@/store';
import { MainLayout } from '@/components/layout/MainLayout';
import { WorkflowsPage } from '@/pages/WorkflowsPage';
import { RunsPage } from '@/pages/RunsPage';
import { HealthPage } from '@/pages/HealthPage';
import { ClientsPage } from '@/pages/ClientsPage';
import { AlertsPage } from '@/pages/AlertsPage';
import { BansPage } from '@/pages/BansPage';
import { NotFoundPage } from '@/pages/NotFoundPage';

const routes = [
  { path: '#/', name: 'Workflows', component: WorkflowsPage },
  { path: '#/workflows', name: 'Workflows', component: WorkflowsPage },
  { path: '#/runs', name: 'Runs', component: RunsPage },
  { path: '#/health', name: 'Health', component: HealthPage },
  { path: '#/clients', name: 'Clients', component: ClientsPage },
  { path: '#/alerts', name: 'Alerts', component: AlertsPage },
  { path: '#/bans', name: 'Bans', component: BansPage },
];

router.register(routes);

function getCurrentPageComponent(path: string) {
  const route = router.getRoute(path);
  if (route) {
    return route.component();
  }
  return NotFoundPage();
}

function renderPage() {
  const app = document.getElementById('app');
  if (!app) return;

  const path = router.getCurrentPath();
  const pageComponent = getCurrentPageComponent(path);
  const layoutComponent = MainLayout(pageComponent);

  app.innerHTML = '';
  app.appendChild(layoutComponent);

  if (store.state.ui.toastMessage) {
    const toast = document.createElement('div');
    toast.className = `p-notification p-notification--${store.state.ui.toastType || 'info'}`;
    toast.textContent = store.state.ui.toastMessage;
    app.appendChild(toast);
  }
}

export function initApp() {
  router.subscribe((path) => {
    store.setCurrentPage(path);
    renderPage();
  });

  renderPage();
}
