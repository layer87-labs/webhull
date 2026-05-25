/**
 * web-core Analytics Collector
 * Lightweight vanilla JS for tracking scroll depth, viewport time, and clicks.
 * Sends events to POST /api/events.
 * Consent-gated: only runs if analytics consent is given.
 */
(function () {
  'use strict';

  var ENDPOINT = '/api/events';
  var DEBOUNCE_MS = 200;
  var SCROLL_THRESHOLDS = [25, 50, 75, 90, 100];
  var scrollReported = {};
  var startTime = Date.now();
  var maxScroll = 0;

  // Check consent before tracking
  function hasConsent() {
    try {
      var cookie = document.cookie.split(';').find(function (c) {
        return c.trim().startsWith('consent=');
      });
      if (!cookie) return false;
      var state = JSON.parse(decodeURIComponent(cookie.trim().substring(8)));
      return state.decided && state.categories && state.categories.analytics;
    } catch (e) {
      return false;
    }
  }

  // Send event to collector endpoint
  function sendEvent(type, props) {
    if (!hasConsent()) return;

    var payload = {
      type: type,
      url: window.location.pathname,
      referrer: document.referrer || '',
      properties: props || {}
    };

    // Use sendBeacon for reliability (especially on page unload)
    if (navigator.sendBeacon) {
      navigator.sendBeacon(ENDPOINT, JSON.stringify(payload));
    } else {
      var xhr = new XMLHttpRequest();
      xhr.open('POST', ENDPOINT, true);
      xhr.setRequestHeader('Content-Type', 'application/json');
      xhr.send(JSON.stringify(payload));
    }
  }

  // Debounce helper
  function debounce(fn, ms) {
    var timer;
    return function () {
      clearTimeout(timer);
      timer = setTimeout(fn, ms);
    };
  }

  // Calculate scroll percentage
  function getScrollPercent() {
    var h = document.documentElement;
    var b = document.body;
    var scrollTop = h.scrollTop || b.scrollTop;
    var scrollHeight = (h.scrollHeight || b.scrollHeight) - h.clientHeight;
    if (scrollHeight <= 0) return 100;
    return Math.round((scrollTop / scrollHeight) * 100);
  }

  // Track scroll depth
  var onScroll = debounce(function () {
    var pct = getScrollPercent();
    if (pct > maxScroll) maxScroll = pct;

    for (var i = 0; i < SCROLL_THRESHOLDS.length; i++) {
      var threshold = SCROLL_THRESHOLDS[i];
      if (pct >= threshold && !scrollReported[threshold]) {
        scrollReported[threshold] = true;
        sendEvent('scroll_depth', { depth: threshold });
      }
    }
  }, DEBOUNCE_MS);

  // Track viewport time on page unload
  function onUnload() {
    var duration = Math.round((Date.now() - startTime) / 1000);
    sendEvent('viewport_time', {
      duration: duration,
      maxScroll: maxScroll
    });
  }

  // Track outbound link clicks
  function onLinkClick(e) {
    var target = e.target.closest('a[href]');
    if (!target) return;

    var href = target.getAttribute('href');
    if (!href) return;

    // Only track external links
    try {
      var url = new URL(href, window.location.origin);
      if (url.origin !== window.location.origin) {
        sendEvent('outbound_click', {
          href: href,
          text: (target.textContent || '').trim().substring(0, 100)
        });
      }
    } catch (e) {
      // Invalid URL, skip
    }
  }

  // Page view event
  function trackPageView() {
    sendEvent('pageview', {
      title: document.title,
      referrer: document.referrer
    });
  }

  // Initialize
  function init() {
    if (!hasConsent()) return;

    trackPageView();
    window.addEventListener('scroll', onScroll, { passive: true });
    window.addEventListener('beforeunload', onUnload);
    document.addEventListener('click', onLinkClick);
  }

  // Run on DOM ready
  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', init);
  } else {
    init();
  }
})();
