/**
 * webhull Cookie Consent Dialog
 *
 * Handles accept / reject / customize interactions and persists the decision
 * via cookie + POST /api/consent.
 *
 * The dialog is a real modal: it takes focus, traps Tab, marks the rest of the
 * page inert, handles Escape and restores focus on close. It can be reopened
 * from any [data-consent-open] trigger so an earlier decision stays
 * withdrawable (GDPR Art. 7(3)).
 *
 * Public API: window.webhullConsent.open() / .close() / .state()
 *
 * Written in ES5 on purpose — this file ships unbundled and untranspiled.
 */
(function () {
  'use strict';

  var ENDPOINT = '/api/consent';
  var COOKIE = 'consent';
  var ANALYTICS = 'analytics';

  var FOCUSABLE = [
    'a[href]',
    'button:not([disabled])',
    'input:not([disabled])',
    'select:not([disabled])',
    'textarea:not([disabled])',
    '[tabindex]:not([tabindex="-1"])'
  ].join(',');

  var banner = document.getElementById('consent-banner');
  if (!banner) return;

  var dialog = banner.querySelector('.consent-dialog');
  var categoriesEl = document.getElementById('consent-categories');
  var actionsInitial = document.getElementById('consent-actions-initial');
  var actionsCustom = document.getElementById('consent-actions-custom');
  var customizeWrap = banner.querySelector('.consent-customize-link');
  var customizeBtn = banner.querySelector('[data-action="customize"]');
  var saveBtn = banner.querySelector('[data-action="save"]');
  var acceptBtn = banner.querySelector('[data-action="accept-all"]');
  var rejectBtn = banner.querySelector('[data-action="reject-all"]');

  // Elements outside the dialog that were made inert while it is open, so the
  // exact previous state can be restored on close.
  var inerted = [];
  var lastFocused = null;
  var isOpen = false;

  // Whether analytics was active in the server-rendered page. A decision that
  // changes this has to reload, because the analytics script tag is emitted
  // server-side based on the consent cookie.
  var analyticsAtLoad = readCookieState().categories[ANALYTICS] === true;

  // ── State ──────────────────────────────────────────────────────────────────

  function readCookieState() {
    var empty = { decided: false, categories: {} };
    try {
      var parts = document.cookie.split(';');
      for (var i = 0; i < parts.length; i++) {
        var part = parts[i].trim();
        if (part.indexOf(COOKIE + '=') !== 0) continue;
        var parsed = JSON.parse(decodeURIComponent(part.substring(COOKIE.length + 1)));
        if (!parsed || typeof parsed !== 'object') return empty;
        return {
          decided: parsed.decided === true,
          categories: parsed.categories || {}
        };
      }
    } catch (e) {
      // Malformed cookie is treated as "nothing decided".
    }
    return empty;
  }

  function getCategoryInputs() {
    return banner.querySelectorAll('input[data-category]');
  }

  // Build a consent state. Pass true/false to force all non-required
  // categories, or nothing to read the current checkbox values.
  function buildState(allAccepted) {
    var categories = {};
    var inputs = getCategoryInputs();
    for (var i = 0; i < inputs.length; i++) {
      var input = inputs[i];
      var key = input.getAttribute('data-category');
      if (allAccepted !== undefined) {
        categories[key] = input.disabled ? true : allAccepted;
      } else {
        categories[key] = input.checked;
      }
    }
    return { decided: true, categories: categories };
  }

  // Persist the decision. The cookie is written first so it is already in
  // place if the page reloads before the async request completes.
  function sendConsent(state) {
    var json = JSON.stringify(state);
    document.cookie = COOKIE + '=' + encodeURIComponent(json) +
      ';path=/;max-age=' + (365 * 24 * 3600) + ';SameSite=Lax;Secure';
    try {
      var xhr = new XMLHttpRequest();
      xhr.open('POST', ENDPOINT, true);
      xhr.setRequestHeader('Content-Type', 'application/json');
      xhr.send(json);
    } catch (e) {
      // The cookie is already set; server sync may fail silently.
    }
  }

  // Persist, close, and reload when the analytics decision changed — the
  // script tag is rendered server-side, so the page has to be re-fetched both
  // to start and to stop client-side tracking.
  function commit(state) {
    sendConsent(state);
    close();
    if ((state.categories[ANALYTICS] === true) !== analyticsAtLoad) {
      window.location.reload();
    }
  }

  // ── Focus management ───────────────────────────────────────────────────────

  function focusableItems() {
    var nodes = dialog.querySelectorAll(FOCUSABLE);
    var out = [];
    for (var i = 0; i < nodes.length; i++) {
      var el = nodes[i];
      if (el.closest('[hidden]')) continue;
      if (!el.getClientRects().length) continue;
      out.push(el);
    }
    return out;
  }

  // Mark everything outside the dialog inert so screen readers and Tab cannot
  // reach the page behind the overlay. aria-modal alone does not do this.
  function deactivateBackground() {
    inerted = [];
    var children = document.body.children;
    for (var i = 0; i < children.length; i++) {
      var el = children[i];
      if (el === banner || el.contains(banner)) continue;
      inerted.push({
        el: el,
        inert: el.inert,
        aria: el.getAttribute('aria-hidden')
      });
      el.inert = true;
      el.setAttribute('aria-hidden', 'true');
    }
  }

  function reactivateBackground() {
    for (var i = 0; i < inerted.length; i++) {
      var entry = inerted[i];
      entry.el.inert = entry.inert;
      if (entry.aria === null) {
        entry.el.removeAttribute('aria-hidden');
      } else {
        entry.el.setAttribute('aria-hidden', entry.aria);
      }
    }
    inerted = [];
  }

  function onKeydown(event) {
    if (event.key === 'Escape' || event.key === 'Esc') {
      event.preventDefault();
      onEscape();
      return;
    }
    if (event.key !== 'Tab') return;

    var items = focusableItems();
    if (!items.length) {
      event.preventDefault();
      dialog.focus();
      return;
    }
    var first = items[0];
    var last = items[items.length - 1];

    // Focus escaped the dialog (browser without inert support, programmatic
    // focus elsewhere) — pull it back in.
    if (!dialog.contains(document.activeElement)) {
      event.preventDefault();
      first.focus();
      return;
    }

    if (event.shiftKey && (document.activeElement === first || document.activeElement === dialog)) {
      event.preventDefault();
      last.focus();
    } else if (!event.shiftKey && document.activeElement === last) {
      event.preventDefault();
      first.focus();
    }
  }

  // Escape on a reopened dialog closes it and leaves the stored decision
  // untouched. Escape on the first-visit dialog is treated as "reject all":
  // leaving the decision unmade would keep the modal up forever, and refusing
  // is the privacy-preserving reading of a dismissal. It mirrors the equally
  // prominent reject button rather than inventing a third outcome.
  function onEscape() {
    if (readCookieState().decided) {
      close();
      return;
    }
    if (rejectBtn) {
      commit(buildState(false));
    } else {
      close();
    }
  }

  // ── Open / close ───────────────────────────────────────────────────────────

  // Reset the dialog body. A first visit starts with the summary view; a
  // reopened dialog starts with the toggles expanded, since changing the
  // details is why it was reopened. The one-click accept/reject shortcuts stay
  // available in both cases.
  function setView(expanded) {
    if (!categoriesEl) return;
    if (expanded) {
      categoriesEl.removeAttribute('hidden');
      if (customizeWrap) customizeWrap.setAttribute('hidden', '');
      if (actionsCustom) actionsCustom.removeAttribute('hidden');
    } else {
      categoriesEl.setAttribute('hidden', '');
      if (customizeWrap) customizeWrap.removeAttribute('hidden');
      if (actionsCustom) actionsCustom.setAttribute('hidden', '');
    }
    if (actionsInitial) actionsInitial.removeAttribute('hidden');
  }

  // Reflect the stored decision on the toggles, so a reopened dialog shows
  // what is actually in effect.
  function syncInputs() {
    var state = readCookieState();
    if (!state.decided) return;
    var inputs = getCategoryInputs();
    for (var i = 0; i < inputs.length; i++) {
      var input = inputs[i];
      if (input.disabled) continue;
      input.checked = state.categories[input.getAttribute('data-category')] === true;
    }
  }

  function open() {
    if (isOpen) return;
    isOpen = true;

    var decided = readCookieState().decided;
    syncInputs();
    setView(decided);

    lastFocused = document.activeElement;
    banner.removeAttribute('hidden');
    document.body.classList.add('consent-open');
    deactivateBackground();
    document.addEventListener('keydown', onKeydown, true);

    // Focus the dialog itself rather than the first button, so the title and
    // description are announced before the available actions.
    if (dialog) dialog.focus();
  }

  function close() {
    if (!isOpen) return;
    isOpen = false;

    document.removeEventListener('keydown', onKeydown, true);
    reactivateBackground();
    banner.setAttribute('hidden', '');
    document.body.classList.remove('consent-open');

    if (lastFocused && document.contains(lastFocused) && lastFocused.focus) {
      lastFocused.focus();
    }
    lastFocused = null;
  }

  // ── Wiring ─────────────────────────────────────────────────────────────────

  if (acceptBtn) {
    acceptBtn.addEventListener('click', function () {
      commit(buildState(true));
    });
  }

  if (rejectBtn) {
    rejectBtn.addEventListener('click', function () {
      commit(buildState(false));
    });
  }

  if (customizeBtn && categoriesEl) {
    customizeBtn.addEventListener('click', function () {
      setView(true);
      var first = categoriesEl.querySelector('input:not([disabled])');
      if (first) first.focus();
    });
  }

  if (saveBtn) {
    saveBtn.addEventListener('click', function () {
      commit(buildState());
    });
  }

  // Any element carrying [data-consent-open] reopens the dialog — used by the
  // footer link, and available to site templates.
  document.addEventListener('click', function (event) {
    var target = event.target;
    if (!target || typeof target.closest !== 'function') return;
    var trigger = target.closest('[data-consent-open]');
    if (!trigger) return;
    event.preventDefault();
    open();
  });

  window.webhullConsent = {
    open: open,
    close: close,
    state: readCookieState
  };

  // Open immediately when the server rendered the dialog without a decision.
  if (!banner.hasAttribute('hidden')) {
    open();
  }
})();
