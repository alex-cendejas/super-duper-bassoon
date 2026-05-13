import { store } from '@/store';
import { CreateWorkflowModal } from './CreateWorkflowModal';
import { ClientDetailModal } from './ClientDetailModal';

const modalRegistry: Record<string, (id?: string) => HTMLElement> = {
  'create-workflow': CreateWorkflowModal,
};

let currentModals: Map<string, HTMLElement> = new Map();

function isClientDetailModal(modalName: string): boolean {
  return modalName.startsWith('client-detail-');
}

function getClientIdFromModalName(modalName: string): string | null {
  if (!isClientDetailModal(modalName)) return null;
  return modalName.replace('client-detail-', '');
}

export function initializeModalRenderer(): void {
  store.subscribe((state) => {
    const openModals = Object.keys(state.ui.modals).filter((key) => state.ui.modals[key]);
    const closedModals = Object.keys(state.ui.modals).filter((key) => !state.ui.modals[key]);

    closedModals.forEach((modalName) => {
      const element = currentModals.get(modalName);
      if (element) {
        element.remove();
        currentModals.delete(modalName);
      }
    });

    openModals.forEach((modalName) => {
      if (!currentModals.has(modalName)) {
        let element: HTMLElement | null = null;

        if (isClientDetailModal(modalName)) {
          const clientId = getClientIdFromModalName(modalName);
          if (clientId) {
            element = ClientDetailModal(clientId);
          }
        } else if (modalRegistry[modalName]) {
          element = modalRegistry[modalName]();
        }

        if (element) {
          document.body.appendChild(element);
          currentModals.set(modalName, element);
        }
      }
    });
  });
}
