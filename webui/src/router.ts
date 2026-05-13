export interface Route {
  path: string;
  name: string;
  component: () => HTMLElement;
}

export class Router {
  private routes: Route[] = [];
  private currentPath = '#';
  private listeners: Set<(path: string) => void> = new Set();

  constructor() {
    window.addEventListener('hashchange', () => {
      this.handleRouteChange();
    });
  }

  register(routes: Route[]): void {
    this.routes = routes;
  }

  navigate(path: string): void {
    window.location.hash = path;
  }

  getCurrentPath(): string {
    const hash = window.location.hash;
    if (!hash || hash === '#') return '#/';
    return hash;
  }

  getRoute(path: string): Route | undefined {
    const normalizedPath = path.split('?')[0];
    return this.routes.find((route) => {
      if (route.path === normalizedPath) return true;
      // Handle parameterized routes
      const routeParts = route.path.split('/');
      const pathParts = normalizedPath.split('/');
      if (routeParts.length !== pathParts.length) return false;
      return routeParts.every((part, i) => part.startsWith(':') || part === pathParts[i]);
    });
  }

  private handleRouteChange(): void {
    const path = this.getCurrentPath();
    if (path !== this.currentPath) {
      this.currentPath = path;
      this.listeners.forEach((listener) => listener(path));
    }
  }

  subscribe(listener: (path: string) => void): () => void {
    this.listeners.add(listener);
    listener(this.getCurrentPath());
    return () => this.listeners.delete(listener);
  }

  getParams(path: string): Record<string, string> {
    const route = this.getRoute(path);
    if (!route) return {};

    const routeParts = route.path.split('/');
    const pathParts = path.split('/');
    const params: Record<string, string> = {};

    routeParts.forEach((part, i) => {
      if (part.startsWith(':')) {
        const paramName = part.slice(1);
        params[paramName] = pathParts[i] || '';
      }
    });

    return params;
  }
}

export const router = new Router();
