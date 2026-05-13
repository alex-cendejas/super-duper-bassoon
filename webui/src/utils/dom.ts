export function qs<T extends Element = Element>(selector: string, parent: Document | Element = document): T | null {
  return parent.querySelector(selector) as T | null;
}

export function qsa<T extends Element = Element>(
  selector: string,
  parent: Document | Element = document
): T[] {
  return Array.from(parent.querySelectorAll(selector)) as T[];
}

export function createElement<K extends keyof HTMLElementTagNameMap>(
  tag: K,
  options?: { class?: string; id?: string; attrs?: Record<string, string> }
): HTMLElementTagNameMap[K] {
  const el = document.createElement(tag);
  if (options?.class) el.className = options.class;
  if (options?.id) el.id = options.id;
  if (options?.attrs) {
    Object.entries(options.attrs).forEach(([key, value]) => {
      el.setAttribute(key, value);
    });
  }
  return el;
}

export function addClass(el: Element, className: string): void {
  el.classList.add(className);
}

export function removeClass(el: Element, className: string): void {
  el.classList.remove(className);
}

export function toggleClass(el: Element, className: string, force?: boolean): void {
  el.classList.toggle(className, force);
}

export function hasClass(el: Element, className: string): boolean {
  return el.classList.contains(className);
}

export function setText(el: Element, text: string): void {
  el.textContent = text;
}

export function setHtml(el: Element, html: string): void {
  el.innerHTML = html;
}

export function on<K extends keyof HTMLElementEventMap>(
  el: Element,
  event: K,
  handler: (this: Element, ev: HTMLElementEventMap[K]) => void,
  options?: boolean | AddEventListenerOptions
): void {
  el.addEventListener(event, handler as EventListener, options);
}

export function off<K extends keyof HTMLElementEventMap>(
  el: Element,
  event: K,
  handler: (this: Element, ev: HTMLElementEventMap[K]) => void,
  options?: boolean | EventListenerOptions
): void {
  el.removeEventListener(event, handler as EventListener, options);
}

export function emptyElement(el: Element): void {
  el.innerHTML = '';
}
