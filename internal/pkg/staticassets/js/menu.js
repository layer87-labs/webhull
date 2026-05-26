/**
 * web-core UI Script
 * Theme toggle, mobile menu, dropdown navigation.
 */
document.addEventListener('DOMContentLoaded', function () {
  'use strict';

  // --- Theme Toggle ---
  var themeToggle = document.querySelector('.theme-toggle');
  if (themeToggle) {
    themeToggle.addEventListener('click', function () {
      var current = document.documentElement.getAttribute('data-theme');
      var next = current === 'light' ? 'dark' : 'light';
      document.documentElement.setAttribute('data-theme', next);
      localStorage.setItem('theme', next);
    });
  }

  // --- Mobile Menu Toggle ---
  var mobileMenuToggle = document.querySelector('.mobile-menu-toggle');
  var navLinks = document.querySelector('.nav-links');

  if (mobileMenuToggle && navLinks) {
    mobileMenuToggle.addEventListener('click', function () {
      navLinks.classList.toggle('active');
      mobileMenuToggle.classList.toggle('active');
    });

    // Close mobile menu when clicking outside
    document.addEventListener('click', function (event) {
      if (!event.target.closest('.nav-container')) {
        navLinks.classList.remove('active');
        mobileMenuToggle.classList.remove('active');
      }
    });
  }

  // --- Mobile Dropdown Toggle ---
  var dropdowns = document.querySelectorAll('.nav-links .dropdown > a');
  dropdowns.forEach(function (dropdown) {
    dropdown.addEventListener('click', function (e) {
      if (window.innerWidth <= 768) {
        e.preventDefault();
        e.stopPropagation();

        var parent = this.parentElement;
        var wasActive = parent.classList.contains('active');

        // Close all other dropdowns
        document.querySelectorAll('.nav-links .dropdown').forEach(function (d) {
          d.classList.remove('active');
        });

        // Toggle current dropdown
        if (!wasActive) {
          parent.classList.add('active');
        }
      }
    });
  });

  // --- Smooth Scroll ---
  document.querySelectorAll('a[href^="#"]').forEach(function (anchor) {
    anchor.addEventListener('click', function (e) {
      var href = this.getAttribute('href');
      if (href !== '#' && href.length > 1) {
        e.preventDefault();
        var target = document.querySelector(href);
        if (target) {
          target.scrollIntoView({ behavior: 'smooth', block: 'start' });
        }
      }
    });
  });
});
