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

  // Per-field error display helpers
  function setFieldError(input, msg) {
    var field = input.closest('.form-field, .contact__field');
    if (!field) return;
    field.classList.add('form-field--error');
    var hint = field.querySelector('.form-field-hint');
    if (!hint) {
      hint = document.createElement('span');
      hint.className = 'form-field-hint';
      hint.setAttribute('role', 'alert');
      field.appendChild(hint);
    }
    hint.textContent = msg;
  }

  function clearFieldErrors() {
    form.querySelectorAll('.form-field--error, .contact__field--error').forEach(function (el) {
      el.classList.remove('form-field--error', 'contact__field--error');
    });
    form.querySelectorAll('.form-field-hint').forEach(function (el) { el.remove(); });
  }

  // Basic email format check
  function isValidEmail(val) {
    return /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(val);
  }

  // Returns true when all fields pass client-side validation
  function validate() {
    var ok = true;
    form.querySelectorAll('.form-input, .form-textarea, .form-select').forEach(function (input) {
      var val = input.value.trim();
      if (input.required && !val) {
        setFieldError(input, 'Pflichtfeld');
        ok = false;
      } else if (input.type === 'email' && val && !isValidEmail(val)) {
        setFieldError(input, 'Bitte eine gültige E-Mail-Adresse eingeben');
        ok = false;
      }
    });
    return ok;
  }

  form.addEventListener('submit', function (e) {
    e.preventDefault();

    // Reset state
    if (successEl) successEl.setAttribute('hidden', '');
    if (errorEl) errorEl.setAttribute('hidden', '');
    clearFieldErrors();

    if (!validate()) return;

    // Collect all form fields dynamically (skip honeypot)
    var fields = {};
    form.querySelectorAll('.form-input, .form-textarea, .form-select').forEach(function (input) {
      if (input.name) fields[input.name] = input.value.trim();
    });
    var data = {
      fields: fields,
      website: (form.querySelector('[name="website"]') || {}).value || ''
    };

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
          if (errorEl) errorEl.removeAttribute('hidden');
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

  // Clear per-field error on input
  form.addEventListener('input', function (e) {
    var field = e.target.closest('.form-field--error, .contact__field--error');
    if (field) {
      field.classList.remove('form-field--error', 'contact__field--error');
      var hint = field.querySelector('.form-field-hint');
      if (hint) hint.remove();
    }
  });
})();
