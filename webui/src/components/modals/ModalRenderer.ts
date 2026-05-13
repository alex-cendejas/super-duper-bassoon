import { store } from '@/store';
import { CreateWorkflowModal } from './CreateWorkflowModal';

const modalRegistry: Record<string, () => HTMLElement> = {
  'create-workflow': CreateWorkflowModal,
};

let currentModals: Map<string, HTMLElement> = new Map();

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
      if (!currentModals.has(modalName) && modalRegistry[modalName]) {
        const element = modalRegistry[modalName]();
        document.body.appendChild(element);
        currentModals.set(modalName, element);
      }
    });
  });
}
