/**
 * web-core Cookie Consent Banner
 * Handles accept/reject/customize interactions.
 * Persists state via POST /api/consent and cookie.
 */
(function () {
  'use strict';

  var ENDPOINT = '/api/consent';
  var banner = document.getElementById('consent-banner');
  if (!banner) return;

  var categoriesEl = document.getElementById('consent-categories');
  var actionsInitial = document.getElementById('consent-actions-initial');
  var actionsCustom = document.getElementById('consent-actions-custom');
  var customizeBtn = banner.querySelector('[data-action="customize"]');
  var saveBtn = banner.querySelector('[data-action="save"]');
  var acceptBtn = banner.querySelector('[data-action="accept-all"]');
  var rejectBtn = banner.querySelector('[data-action="reject-all"]');

  // Get all category checkboxes
  function getCategoryInputs() {
    return banner.querySelectorAll('input[data-category]');
  }

  // Build state from current checkbox values
  function buildState(allAccepted) {
    var categories = {};
    var inputs = getCategoryInputs();
    for (var i = 0; i < inputs.length; i++) {
      var input = inputs[i];
      var key = input.getAttribute('data-category');
      if (allAccepted !== undefined) {
        // For accept/reject all: set all non-required to the value
        categories[key] = input.disabled ? true : allAccepted;
      } else {
        categories[key] = input.checked;
      }
    }
    return { decided: true, categories: categories };
  }

  // Send consent state to server and set cookie client-side.
  // Cookie is set immediately so it survives a page reload that
  // may happen before the async XHR response arrives.
  function sendConsent(state) {
    var json = JSON.stringify(state);
    // Set cookie client-side first (available instantly on reload)
    document.cookie = 'consent=' + encodeURIComponent(json) +
      ';path=/;max-age=' + (365 * 24 * 3600) + ';SameSite=Lax;Secure';
    // Notify server (best-effort, non-blocking)
    try {
      var xhr = new XMLHttpRequest();
      xhr.open('POST', ENDPOINT, true);
      xhr.setRequestHeader('Content-Type', 'application/json');
      xhr.send(json);
    } catch (e) {
      // Cookie is already set, server sync can fail silently
    }
  }

  // Hide the banner
  function hideBanner() {
    banner.setAttribute('hidden', '');
    banner.style.display = 'none';
    document.body.classList.remove('consent-open');
  }

  // Accept all
  if (acceptBtn) {
    acceptBtn.addEventListener('click', function () {
      var state = buildState(true);
      sendConsent(state);
      hideBanner();
      // Reload to activate analytics scripts
      window.location.reload();
    });
  }

  // Reject all (only required categories stay on)
  if (rejectBtn) {
    rejectBtn.addEventListener('click', function () {
      var state = buildState(false);
      sendConsent(state);
      hideBanner();
    });
  }

  // Show customize view: reveal categories + save, hide initial buttons
  if (customizeBtn && categoriesEl) {
    customizeBtn.addEventListener('click', function () {
      categoriesEl.removeAttribute('hidden');
      customizeBtn.parentElement.setAttribute('hidden', '');
      if (actionsInitial) actionsInitial.setAttribute('hidden', '');
      if (actionsCustom) actionsCustom.removeAttribute('hidden');
    });
  }

  // Save custom preferences
  if (saveBtn) {
    saveBtn.addEventListener('click', function () {
      var state = buildState();
      sendConsent(state);
      hideBanner();
      // Reload if analytics was enabled
      if (state.categories.analytics) {
        window.location.reload();
      }
    });
  }

  // Add body class for styling when banner is visible
  document.body.classList.add('consent-open');
})();
