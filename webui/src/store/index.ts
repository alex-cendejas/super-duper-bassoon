import type {
  Workflow,
  Run,
  ClientMetadata,
  TypeHealth,
  BanRecord,
  Alert,
  SystemStatus,
} from '@/types';

export interface AppState {
  workflows: {
    items: Workflow[];
    loading: boolean;
    error?: string;
  };
  runs: {
    items: Run[];
    total: number;
    page: number;
    limit: number;
    loading: boolean;
    error?: string;
  };
  clients: {
    items: ClientMetadata[];
    loading: boolean;
    error?: string;
  };
  health: {
    items: TypeHealth[];
    loading: boolean;
    error?: string;
  };
  bans: {
    items: BanRecord[];
    loading: boolean;
    error?: string;
  };
  alerts: {
    items: Alert[];
    total: number;
    page: number;
    limit: number;
    loading: boolean;
    error?: string;
  };
  system: {
    status?: SystemStatus;
    loading: boolean;
    error?: string;
  };
  ui: {
    sidebarOpen: boolean;
    currentPage: string;
    modals: Record<string, boolean>;
    toastMessage?: string;
    toastType?: 'success' | 'error' | 'info';
  };
}

export class Store {
  state: AppState;
  private listeners: Set<(state: AppState) => void> = new Set();

  constructor() {
    this.state = {
      workflows: { items: [], loading: false },
      runs: { items: [], total: 0, page: 1, limit: 10, loading: false },
      clients: { items: [], loading: false },
      health: { items: [], loading: false },
      bans: { items: [], loading: false },
      alerts: { items: [], total: 0, page: 1, limit: 25, loading: false },
      system: { loading: false },
      ui: { sidebarOpen: true, currentPage: '#', modals: {} },
    };
  }

  subscribe(listener: (state: AppState) => void): () => void {
    this.listeners.add(listener);
    listener(this.state);
    return () => this.listeners.delete(listener);
  }

  notify(): void {
    this.listeners.forEach((listener) => listener(this.state));
  }

  setState<K extends keyof AppState>(slice: K, updates: Partial<AppState[K]>): void {
    this.state[slice] = { ...this.state[slice], ...updates };
    this.notify();
  }

  setWorkflows(items: Workflow[]): void {
    this.setState('workflows', { items, loading: false });
  }

  setWorkflowLoading(loading: boolean, error?: string): void {
    this.setState('workflows', { loading, error });
  }

  setRuns(response: { items: Run[]; total: number }): void {
    this.setState('runs', {
      items: response.items,
      total: response.total,
      loading: false,
    });
  }

  setRunsLoading(loading: boolean, error?: string): void {
    this.setState('runs', { loading, error });
  }

  setClients(items: ClientMetadata[]): void {
    this.setState('clients', { items, loading: false });
  }

  setClientsLoading(loading: boolean, error?: string): void {
    this.setState('clients', { loading, error });
  }

  setHealth(items: TypeHealth[]): void {
    this.setState('health', { items, loading: false });
  }

  setHealthLoading(loading: boolean, error?: string): void {
    this.setState('health', { loading, error });
  }

  setBans(items: BanRecord[]): void {
    this.setState('bans', { items, loading: false });
  }

  setBansLoading(loading: boolean, error?: string): void {
    this.setState('bans', { loading, error });
  }

  setAlerts(response: { items: Alert[]; total: number }): void {
    this.setState('alerts', {
      items: response.items,
      total: response.total,
      loading: false,
    });
  }

  setAlertsLoading(loading: boolean, error?: string): void {
    this.setState('alerts', { loading, error });
  }

  setSystemStatus(status: SystemStatus): void {
    this.setState('system', { status, loading: false });
  }

  setSystemLoading(loading: boolean, error?: string): void {
    this.setState('system', { loading, error });
  }

  openModal(modalName: string): void {
    this.setState('ui', { modals: { ...this.state.ui.modals, [modalName]: true } });
  }

  closeModal(modalName: string): void {
    this.setState('ui', { modals: { ...this.state.ui.modals, [modalName]: false } });
  }

  setCurrentPage(page: string): void {
    this.setState('ui', { currentPage: page });
  }

  toggleSidebar(): void {
    this.setState('ui', { sidebarOpen: !this.state.ui.sidebarOpen });
  }

  showToast(message: string, type: 'success' | 'error' | 'info' = 'info'): void {
    this.setState('ui', { toastMessage: message, toastType: type });
    setTimeout(() => {
      if (this.state.ui.toastMessage === message) {
        this.setState('ui', { toastMessage: undefined });
      }
    }, 4000);
  }
}

export const store = new Store();
