import { mount } from "@cloudflare/nimbus-docs/client";

const BAND_TOP = 0.1;
const BAND_BOTTOM = 0.3;
const SUPPRESS_MS = 1000;

function prefersReducedMotion(): boolean {
  return window.matchMedia("(prefers-reduced-motion: reduce)").matches;
}

function initMobileToc(root: HTMLElement): () => void {
  const select = root.querySelector<HTMLSelectElement>(
    "[data-nb-mobile-toc-select]",
  );
  if (!select) return () => {};

  type Heading = { slug: string; el: HTMLElement };
  const headings: Heading[] = Array.from(select.options)
    .map((o) => o.value)
    .filter((v) => v !== "_top")
    .map((slug) => ({ slug, el: document.getElementById(slug) }))
    .filter((h): h is Heading => h.el !== null);

  const controller = new AbortController();

  let suppress = false;
  let suppressTimer: ReturnType<typeof setTimeout> | undefined;

  function setActive(slug: string) {
    if (select!.value !== slug) select!.value = slug;
  }

  // select → page
  select.addEventListener(
    "change",
    () => {
      const slug = select.value;
      suppress = true;
      clearTimeout(suppressTimer);
      suppressTimer = setTimeout(() => {
        suppress = false;
        resolve();
      }, SUPPRESS_MS);

      const behavior: ScrollBehavior = prefersReducedMotion()
        ? "auto"
        : "smooth";
      if (slug === "_top") {
        window.scrollTo({ top: 0, behavior });
        return;
      }
      document.getElementById(slug)?.scrollIntoView({ behavior });
    },
    { signal: controller.signal },
  );

  if (headings.length === 0) {
    return () => {
      controller.abort();
      clearTimeout(suppressTimer);
    };
  }

  function resolve() {
    if (suppress) return;

    const bandTop = window.innerHeight * BAND_TOP;
    const bandBottom = Math.max(
      window.innerHeight * BAND_BOTTOM,
      parseFloat(getComputedStyle(headings[0].el).scrollMarginTop),
    );
    let active = "_top";
    for (const { slug, el } of headings) {
      const rect = el.getBoundingClientRect();
      if (rect.top > bandBottom) break;
      active = slug;
      if (rect.bottom >= bandTop) break;
    }
    setActive(active);
  }

  let frame = 0;
  function onScroll() {
    if (frame) return;
    frame = requestAnimationFrame(() => {
      frame = 0;
      resolve();
    });
  }

  for (const event of ["scroll", "resize", "pageshow"]) {
    window.addEventListener(event, onScroll, {
      passive: true,
      signal: controller.signal,
    });
  }
  resolve();

  return () => {
    controller.abort();
    cancelAnimationFrame(frame);
    clearTimeout(suppressTimer);
  };
}

mount("[data-nb-mobile-toc]", initMobileToc);
