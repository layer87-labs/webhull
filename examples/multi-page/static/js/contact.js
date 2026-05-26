/**
 * web-core Contact Form Handler
 * Progressive enhancement: form works without JS (standard POST),
 * but with JS it submits via fetch for a better UX.
 */
(function () {
  'use strict';

  var form = document.getElementById('contact-form');
  if (!form) return;

  var successEl = document.getElementById('contact-success');
  var errorEl = document.getElementById('contact-error');
  var submitBtn = document.getElementById('contact-submit');
  var originalBtnText = submitBtn ? submitBtn.textContent : '';

  form.addEventListener('submit', function (e) {
    e.preventDefault();

    // Hide previous messages
    if (successEl) successEl.setAttribute('hidden', '');
    if (errorEl) errorEl.setAttribute('hidden', '');

    // Collect form data
    var data = {
      name: form.querySelector('[name="name"]').value.trim(),
      email: form.querySelector('[name="email"]').value.trim(),
      subject: form.querySelector('[name="subject"]').value.trim(),
      message: form.querySelector('[name="message"]').value.trim(),
      website: form.querySelector('[name="website"]').value // honeypot
    };

    // Client-side validation
    if (!data.name || !data.email || !data.subject || !data.message) {
      if (errorEl) {
        errorEl.removeAttribute('hidden');
      }
      return;
    }

    // Disable button and show loading state
    if (submitBtn) {
      submitBtn.disabled = true;
      submitBtn.textContent = '...';
    }

    // Submit via fetch
    fetch(form.getAttribute('action'), {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(data)
    })
      .then(function (res) {
        return res.json().then(function (body) {
          return { ok: res.ok, body: body };
        });
      })
      .then(function (result) {
        if (result.ok && result.body.success) {
          if (successEl) successEl.removeAttribute('hidden');
          form.reset();
          form.style.display = 'none';
        } else {
          if (errorEl) {
            // Show server error message if available
            if (result.body.message && result.body.message !== 'invalid request') {
              errorEl.querySelector('p').textContent = result.body.message;
            }
            errorEl.removeAttribute('hidden');
          }
        }
      })
      .catch(function () {
        if (errorEl) errorEl.removeAttribute('hidden');
      })
      .finally(function () {
        if (submitBtn) {
          submitBtn.disabled = false;
          submitBtn.textContent = originalBtnText;
        }
      });
  });
})();
