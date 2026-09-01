// Initialises a Sentry-compatible browser SDK (Sentry, GlitchTip, ...).
//
// A separate file rather than an inline snippet on purpose: webhull's own CSP
// does not allow 'unsafe-inline' for script-src, so an inline initialiser
// would be blocked by the very policy this file helps report on.
//
// Configuration arrives as data attributes on this script tag, so nothing
// about the target is compiled into webhull.
(function () {
  var el = document.currentScript;
  if (!el) return;

  var dsn = el.getAttribute('data-dsn');
  if (!dsn) return;

  // The SDK is loaded by a separate tag before this one. If it did not arrive
  // — blocked, offline, wrong URL — say so once and stop. Failing loudly in
  // the console beats a page that silently reports nothing.
  if (typeof window.Sentry === 'undefined') {
    console.warn('[webhull] error tracking configured but the SDK did not load');
    return;
  }

  var rate = parseFloat(el.getAttribute('data-sample-rate'));
  if (isNaN(rate) || rate < 0 || rate > 1) rate = 1.0;

  window.Sentry.init({
    dsn: dsn,
    environment: el.getAttribute('data-environment') || undefined,
    release: el.getAttribute('data-release') || undefined,
    sampleRate: rate,

    // No performance tracing and no session replay. This channel exists to
    // catch what server-side tracing cannot see — exceptions in the browser —
    // not to become a second analytics product.
    tracesSampleRate: 0,

    // Never attach IP address, cookies or request headers. Error tracking
    // that carries personal data is a different legal question than error
    // tracking that does not, and webhull ships the answer that needs no
    // consent banner.
    sendDefaultPii: false,

    // Browser extensions are the largest single source of noise in any
    // real-world deployment. Reports whose stack frames live entirely in an
    // extension say nothing about this site.
    beforeSend: function (event) {
      try {
        var frames =
          (event.exception &&
            event.exception.values &&
            event.exception.values[0] &&
            event.exception.values[0].stacktrace &&
            event.exception.values[0].stacktrace.frames) || [];
        var fromExtension = frames.some(function (f) {
          return (
            f.filename &&
            (f.filename.indexOf('chrome-extension://') === 0 ||
              f.filename.indexOf('moz-extension://') === 0 ||
              f.filename.indexOf('safari-extension://') === 0 ||
              f.filename.indexOf('safari-web-extension://') === 0)
          );
        });
        if (fromExtension) return null;
      } catch (e) {
        // A filter that throws must not swallow the event it was filtering.
      }
      return event;
    },
  });
})();
