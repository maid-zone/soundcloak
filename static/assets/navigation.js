(() => {
  /**
   * @param {string} selector
   * @returns {HTMLElement | null}
   */
  const $ = (selector, on = document) => on.querySelector(selector);

  /**
   * @param {string} selector
   * @returns {HTMLElement[] | null}
   */
  const $$ = (selector, on = document) => on.querySelectorAll(selector);

  const pages = new Map();
  const pending = new Map();

  const libraries = new Set(
    [...$$("script[data-library]")].map(
      (s) => /** @type {HTMLScriptElement} */ (s).src,
    ),
  );

  const ttl = 30000;

  let sequence = 0;
  let committing = false;
  let /** @type {number} */ hoverTimer;
  let page = new AbortController();
  let renderedURL = new URL(location.href);

  window.soundcloakPageSignal = page.signal;

  let nextScrollKey = 0;
  const scrollPositions = new Map();
  const positionKey = (state, href) => `${state.scrollKey}:${href}`;
  let restoringHistory = false;

  history.replaceState(
    {
      ...history.state,
      soundcloak: true,
      scrollKey: nextScrollKey++,
      scroll: [scrollX, scrollY],
    },
    "",
  );
  history.scrollRestoration = "manual";
  let currentPositionKey = positionKey(history.state, location.href);
  scrollPositions.set(currentPositionKey, [scrollX, scrollY]);

  window.addEventListener(
    "scroll",
    () => {
      if (!committing && !restoringHistory) {
        scrollPositions.set(currentPositionKey, [scrollX, scrollY]);
      }
    },
    { passive: true },
  );

  /** @param {URL} url  */
  function eligible(url) {
    return (
      url.origin === location.origin &&
      /^https?:$/.test(url.protocol) &&
      !url.pathname.startsWith("/_") &&
      !/\.[a-z0-9]+$/i.test(url.pathname) &&
      !url.searchParams.has("autoplay")
    );
  }

  /** @param {Event} event  */
  function link(event) {
    /** @type {HTMLAnchorElement | null} */
    const a = /** @type {Element} */ (event.target).closest("a[href]");

    if (
      !a ||
      a.hasAttribute("download") ||
      a.hasAttribute("onclick") ||
      a.hasAttribute("data-no-navigation") ||
      a.relList.contains("external") ||
      (a.target && a.target !== "_self")
    )
      return null;
    const url = new URL(a.href);
    if (
      !eligible(url) ||
      url.hash ||
      (url.pathname === location.pathname && url.search === location.search)
    )
      return null;
    return url;
  }

  /** @param {URL} url  */
  async function getPage(url) {
    const key = url.href;
    const cached = pages.get(key);

    if (cached && Date.now() - cached.time < ttl) {
      pages.delete(key);
      pages.set(key, cached);
      return cached;
    }

    pages.delete(key);

    if (pending.has(key)) {
      return pending.get(key);
    }

    const request = (async () => {
      const response = await fetch(key, {
        credentials: "same-origin",
        headers: { Accept: "text/html" },
        signal: AbortSignal.timeout(15000),
      });

      if (
        !response.ok ||
        !eligible(new URL(response.url)) ||
        !response.headers.get("Content-Type")?.includes("text/html")
      ) {
        throw Error("not a page");
      }

      const html = await response.text();

      if (html.length > 1000000) {
        throw Error("page too large");
      }

      const result = {
        html,
        url: response.url,
        time: Date.now(),
      };

      if (!response.headers.get("Cache-Control")?.includes("no-store")) {
        pages.set(key, result);
        while (pages.size > 12) pages.delete(pages.keys().next().value);
      }

      return result;
    })();

    pending.set(key, request);

    try {
      return await request;
    } finally {
      pending.delete(key);
    }
  }

  /** @param {URL | null} url  */
  function prefetch(url) {
    const connection = navigator.connection;

    if (
      !url ||
      document.hidden ||
      connection?.saveData ||
      /(^|-)2g$/.test(connection?.effectiveType || "") ||
      pending.size >= 2
    ) {
      return;
    }

    getPage(url).catch(() => {});
  }
  document.addEventListener("pointerover", (event) => {
    clearTimeout(hoverTimer);
    const url = link(event);
    hoverTimer = setTimeout(() => prefetch(url), 80);
  });

  document.addEventListener("pointerout", () => clearTimeout(hoverTimer));
  document.addEventListener("focusin", (event) => prefetch(link(event)));
  document.addEventListener(
    "pointerdown",
    (event) => {
      if (event.button === 0) prefetch(link(event));
    },
    { passive: true },
  );

  /**
   * @param {HTMLScriptElement} source
   */
  function runScript(source) {
    if (source.hasAttribute("data-navigation")) {
      return Promise.resolve();
    }
    if (source.hasAttribute("data-library") && libraries.has(source.src)) {
      return Promise.resolve();
    }

    // @ts-expect-error
    const initialize = window.soundcloakInitializers?.get(source.src);

    if (initialize) {
      initialize();
      return Promise.resolve();
    }

    return new Promise((resolve, reject) => {
      const script = document.createElement("script");

      for (const attr of source.attributes) {
        script.setAttribute(attr.name, attr.value);
      }

      script.async = false;
      script.removeAttribute("defer");
      script.textContent = source.textContent;

      const timeout = script.src
        ? setTimeout(() => reject(Error("Script timeout")), 10000)
        : null;

      script.onload = () => {
        clearTimeout(timeout);
        if (script.hasAttribute("data-library")) libraries.add(script.src);
        resolve();
      };

      script.onerror = (error) => {
        clearTimeout(timeout);
        reject(error);
      };

      document.body.append(script);

      if (!script.src) resolve();
    });
  }

  /**
   *
   * @param {URL} url
   * @param {object | null} restoring
   * @returns
   */
  async function navigate(url, restoring = null) {
    if (committing) {
      location.assign(url.href);
      return;
    }

    const id = ++sequence;
    const start = performance.now();
    document.documentElement.setAttribute("aria-busy", "true");

    try {
      const result = await getPage(url);

      if (id !== sequence) return;

      const doc = new DOMParser().parseFromString(result.html, "text/html");

      /** @param {Document} d */
      const styles = (d) =>
        [...$$('link[rel="stylesheet"]', d)]
          .map((s) => /** @type {HTMLAnchorElement} */ (s).href)
          .join("|");

      if (
        !$("script[data-navigation]", doc) ||
        styles(doc) !== styles(document) ||
        $("script[data-navigation]", doc)?.src !==
          $("script[data-navigation]")?.src
      ) {
        throw Error("document changed");
      }
      committing = true;
      renderedURL = new URL(result.url);
      if (!restoring) {
        scrollPositions.set(currentPositionKey, [scrollX, scrollY]);
        history.replaceState(
          { ...history.state, scroll: [scrollX, scrollY] },
          "",
        );
        history.pushState(
          { soundcloak: true, scrollKey: nextScrollKey++, scroll: [0, 0] },
          "",
          result.url,
        );
      }
      currentPositionKey = positionKey(history.state, location.href);
      page.abort();
      page = new AbortController();

      window.soundcloakPageSignal = page.signal;
      const scripts = [...$$("script", doc)];

      for (const script of scripts) {
        script.remove();
      }

      document.title = doc.title;

      for (const e of $$(
        'meta[name^="og:"], link[rel="icon"]',
        document.head,
      )) {
        e.remove();
      }

      for (const e of $$('meta[name^="og:"], link[rel="icon"]', doc.head)) {
        document.head.append(e);
      }

      document.body.replaceWith(doc.body);
      document.body.classList.remove("noJs");

      for (const script of scripts) {
        await runScript(script);
      }

      /** @type {HTMLElement | null} */
      const focus = $(".hero-section h1, h1, .header-search-query");

      if (focus && !restoring) {
        focus.setAttribute("tabindex", "-1");
        focus.focus({ preventScroll: true });
      }

      scrollTo(...(restoring?.scroll || [0, 0]));
      scrollPositions.set(currentPositionKey, [scrollX, scrollY]);

      performance.measure("soundcloak:navigation", {
        start,
        end: performance.now(),
      });

      // keep bounded during long browsing sessions
      if (performance.getEntriesByName("soundcloak:navigation").length > 100) {
        performance.clearMeasures("soundcloak:navigation");
      }
    } catch {
      if (id === sequence) {
        location.assign(url.href);
      }
    } finally {
      if (id === sequence) {
        committing = false;
        restoringHistory = false;
        document.documentElement.removeAttribute("aria-busy");
      }
    }
  }

  document.addEventListener("click", (event) => {
    if (
      event.defaultPrevented ||
      event.button !== 0 ||
      event.metaKey ||
      event.ctrlKey ||
      event.shiftKey ||
      event.altKey
    )
      return;

    const url = link(event);

    if (!url) return;

    event.preventDefault();
    navigate(url);
  });

  document.addEventListener("submit", (event) => {
    const form = /** @type {HTMLFormElement} */ (event.target);
    if (!form.matches(".header-search") || event.defaultPrevented) return;
    const url = new URL(form.action);
    url.search = new URLSearchParams(new FormData(form)).toString();
    event.preventDefault();
    navigate(url);
  });

  window.addEventListener("popstate", (event) => {
    if (!event.state?.soundcloak || !eligible(new URL(location.href))) {
      location.reload();
      return;
    }
    const destinationKey = positionKey(event.state, location.href);
    const restoring = {
      ...event.state,
      scroll: scrollPositions.get(destinationKey) || event.state.scroll,
    };
    restoringHistory = true;
    if (
      !committing &&
      location.pathname === renderedURL.pathname &&
      location.search === renderedURL.search
    ) {
      ++sequence;
      document.documentElement.removeAttribute("aria-busy");
      let target;
      try {
        target =
          location.hash &&
          document.getElementById(decodeURIComponent(location.hash.slice(1)));
      } catch {
        // malformed fragment
      }
      if (target) target.scrollIntoView();
      else scrollTo(...(restoring.scroll || [0, 0]));
      currentPositionKey = destinationKey;
      scrollPositions.set(currentPositionKey, [scrollX, scrollY]);
      restoringHistory = false;
      return;
    }
    navigate(new URL(location.href), restoring);
  });

  // preferences use a full document navigation, discard pre-preference html if
  // this document is subsequently restored from the browser's back/forward cache
  window.addEventListener("pageshow", (event) => {
    if (event.persisted) {
      pages.clear();
      location.reload();
    }
  });
})();
