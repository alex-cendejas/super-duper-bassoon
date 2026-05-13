import { Header } from './Header';
import { Sidebar } from './Sidebar';
import { store } from '@/store';

export function MainLayout(content: HTMLElement): HTMLElement {
  const container = document.createElement('div');
  container.className = 'main-layout';

  const header = Header();
  const sidebar = Sidebar();

  const mainContent = document.createElement('main');
  mainContent.className = 'p-layout__main';
  mainContent.appendChild(content);

  container.appendChild(header);

  const layoutContainer = document.createElement('div');
  layoutContainer.className = 'p-layout';
  layoutContainer.appendChild(sidebar);
  layoutContainer.appendChild(mainContent);

  container.appendChild(layoutContainer);

  store.subscribe((state) => {
    if (state.ui.sidebarOpen) {
      sidebar.classList.remove('is-hidden');
    } else {
      sidebar.classList.add('is-hidden');
    }
  });

  return container;
}
