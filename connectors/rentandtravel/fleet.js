// rentandtravel fleet detail dialog — gallery, specs, availability calendar.
//
// Ships as an external file, not inline in fleet.tmpl.html, deliberately:
// webhull's default CSP is script-src 'self' [+ analytics host], no
// 'unsafe-inline' and no nonce. An inline <script> in the plugin fragment
// is silently blocked by the browser — no exception is thrown that a
// server-side render or a CDP-injected test script would ever see, so it
// can look like it works in dev and be completely dead in production. See
// this connector's README for the required setup step: copy this file to
// the consuming site's static directory and serve it same-origin.
(function () {
  "use strict";
  var dialog = document.getElementById("fleet-detail-dialog");
  if (!dialog || typeof dialog.showModal !== "function") return;

  var img = document.getElementById("fleet-dialog-img");
  var thumbs = document.getElementById("fleet-dialog-thumbs");
  var labels = document.getElementById("fleet-dialog-labels");
  var title = document.getElementById("fleet-dialog-title");
  var group = document.getElementById("fleet-dialog-group");
  var stats = document.getElementById("fleet-dialog-stats");
  var features = document.getElementById("fleet-dialog-features");
  var intro = document.getElementById("fleet-dialog-intro");
  var techHead = document.getElementById("fleet-dialog-tech-head");
  var tech = document.getElementById("fleet-dialog-tech");
  var calHead = document.getElementById("fleet-dialog-cal-head");
  var cal = document.getElementById("fleet-cal");
  var calMonth = document.getElementById("fleet-cal-month");
  var calGrid = document.getElementById("fleet-cal-grid");
  var calPrev = document.getElementById("fleet-cal-prev");
  var calNext = document.getElementById("fleet-cal-next");
  var price = document.getElementById("fleet-dialog-price");
  var book = document.getElementById("fleet-dialog-book");

  var MONTH_NAMES = ["Januar", "Februar", "März", "April", "Mai", "Juni",
    "Juli", "August", "September", "Oktober", "November", "Dezember"];

  // Parsed with the (year, month, day) constructor — not `new Date(str)` —
  // so the calendar's calendar-day arithmetic is unambiguous regardless of
  // the viewer's timezone offset.
  function parseISODate(s) {
    var parts = s.split("-");
    return new Date(Number(parts[0]), Number(parts[1]) - 1, Number(parts[2]));
  }
  function toISODate(d) {
    var m = String(d.getMonth() + 1).padStart(2, "0");
    var day = String(d.getDate()).padStart(2, "0");
    return d.getFullYear() + "-" + m + "-" + day;
  }
  function sameMonth(a, b) { return a.getFullYear() === b.getFullYear() && a.getMonth() === b.getMonth(); }

  // calState holds the currently-open vehicle's availability data plus
  // which month is on screen, so calPrev/calNext can re-render without
  // needing the vehicle object passed back in.
  var calState = null;

  function renderCalendarMonth() {
    if (!calState) return;
    var shown = calState.shownMonth;
    calMonth.textContent = MONTH_NAMES[shown.getMonth()] + " " + shown.getFullYear();

    var firstOfMonth = new Date(shown.getFullYear(), shown.getMonth(), 1);
    var daysInMonth = new Date(shown.getFullYear(), shown.getMonth() + 1, 0).getDate();
    // getDay(): 0=Sun..6=Sat. Convert to a Monday-first offset (0=Mon..6=Sun).
    var leadingBlanks = (firstOfMonth.getDay() + 6) % 7;

    clear(calGrid);
    for (var b = 0; b < leadingBlanks; b++) {
      var blank = document.createElement("span");
      blank.className = "fleet-cal-day fleet-cal-day--blank";
      calGrid.appendChild(blank);
    }
    for (var day = 1; day <= daysInMonth; day++) {
      var d = new Date(shown.getFullYear(), shown.getMonth(), day);
      var iso = toISODate(d);
      var cell = document.createElement("span");
      cell.className = "fleet-cal-day " + calDayClass(iso);
      cell.textContent = String(day);
      var minNights = calState.minNightsMap[iso];
      if (minNights != null) cell.title = "Mindestens " + minNights + " Nächte";
      calGrid.appendChild(cell);
    }

    calPrev.disabled = sameMonth(shown, calState.minMonth) || shown < calState.minMonth;
    calNext.disabled = sameMonth(shown, calState.maxMonth) || shown > calState.maxMonth;
  }

  function calDayClass(iso) {
    if (!calState.availableSet.has(iso)) return "fleet-cal-day--unavailable";
    var noPickup = calState.disabledPickupSet.has(iso);
    var noReturn = calState.disabledReturnSet.has(iso);
    if (noPickup && noReturn) return "fleet-cal-day--no-pickup fleet-cal-day--no-return";
    if (noPickup) return "fleet-cal-day--no-pickup";
    if (noReturn) return "fleet-cal-day--no-return";
    return "fleet-cal-day--available";
  }

  function setupCalendar(vehicle) {
    var availableDays = vehicle.availableDays || [];
    if (!vehicle.minDate || !vehicle.maxDate || availableDays.length === 0) {
      cal.hidden = true;
      calHead.hidden = true;
      calState = null;
      return;
    }

    calState = {
      availableSet: new Set(availableDays),
      disabledPickupSet: new Set(vehicle.disabledPickup || []),
      disabledReturnSet: new Set(vehicle.disabledReturn || []),
      minNightsMap: vehicle.minNightsMap || {},
      minMonth: parseISODate(vehicle.minDate),
      maxMonth: parseISODate(vehicle.maxDate),
      shownMonth: parseISODate(vehicle.minDate)
    };
    calState.minMonth.setDate(1);
    calState.maxMonth.setDate(1);
    calState.shownMonth.setDate(1);

    cal.hidden = false;
    calHead.hidden = false;
    renderCalendarMonth();
  }

  calPrev.addEventListener("click", function () {
    if (!calState) return;
    calState.shownMonth = new Date(calState.shownMonth.getFullYear(), calState.shownMonth.getMonth() - 1, 1);
    renderCalendarMonth();
  });
  calNext.addEventListener("click", function () {
    if (!calState) return;
    calState.shownMonth = new Date(calState.shownMonth.getFullYear(), calState.shownMonth.getMonth() + 1, 1);
    renderCalendarMonth();
  });

  function isRenderableImage(url) {
    return !!url && !/\.tif$/i.test(url);
  }

  function galleryImages(vehicle) {
    var all = vehicle["images.all"] || [];
    var seen = {};
    var out = [];
    all.forEach(function (entry) {
      var medium = entry.medium;
      if (!isRenderableImage(medium) || seen[medium]) return;
      seen[medium] = true;
      out.push(entry);
    });
    if (out.length === 0) {
      var fallback = vehicle["images.outside.medium"];
      if (isRenderableImage(fallback)) out.push({ small: fallback, medium: fallback, large: fallback });
    }
    return out;
  }

  function clear(el) { while (el.firstChild) el.removeChild(el.firstChild); }

  function openDialog(vehicle) {
    var images = galleryImages(vehicle);

    img.src = images.length ? (images[0].large || images[0].medium) : "";
    img.alt = vehicle.make + " " + vehicle.model;

    clear(thumbs);
    images.forEach(function (entry, idx) {
      var t = document.createElement("button");
      t.type = "button";
      t.className = "fleet-dialog-thumb" + (idx === 0 ? " is-active" : "");
      var thumbImg = document.createElement("img");
      thumbImg.src = entry.small || entry.medium;
      thumbImg.alt = "";
      thumbImg.loading = "lazy";
      t.appendChild(thumbImg);
      t.addEventListener("click", function () {
        img.src = entry.large || entry.medium;
        thumbs.querySelectorAll(".fleet-dialog-thumb").forEach(function (x) { x.classList.remove("is-active"); });
        t.classList.add("is-active");
      });
      thumbs.appendChild(t);
    });
    thumbs.hidden = images.length <= 1;

    clear(labels);
    (vehicle.labels || []).forEach(function (l) {
      var span = document.createElement("span");
      span.className = "fleet-badge";
      span.textContent = l;
      labels.appendChild(span);
    });

    title.textContent = vehicle.make + " " + vehicle.model;
    group.textContent = vehicle.group || "";

    // Icon + value + label, deliberately not icon-only: an icon alone has
    // no accessible name for a screen reader, and has to be decoded before
    // it means anything. Value stays the visually dominant element so the
    // strip is still scannable at a glance.
    clear(stats);
    [
      ["f-seats", vehicle.seats, "Sitzplätze"],
      ["f-beds", vehicle.beds, "Schlafplätze"],
      ["f-licence", vehicle.driversLicence, "Führerschein"],
      ["f-year", vehicle.modelYear, "Baujahr"],
      ["f-pets", vehicle.petsAllowed == null ? null : (vehicle.petsAllowed ? "Ja" : "Nein"), "Haustiere"]
    ].forEach(function (row) {
      var icon = row[0], value = row[1], label = row[2];
      if (value == null || value === "") return;

      var li = document.createElement("li");
      li.className = "fleet-stat";

      var svg = document.createElementNS("http://www.w3.org/2000/svg", "svg");
      svg.setAttribute("class", "fleet-stat-icon");
      svg.setAttribute("aria-hidden", "true");
      svg.setAttribute("focusable", "false");
      var use = document.createElementNS("http://www.w3.org/2000/svg", "use");
      use.setAttribute("href", "#" + icon);
      svg.appendChild(use);

      var text = document.createElement("span");
      text.className = "fleet-stat-text";
      var v = document.createElement("strong");
      v.className = "fleet-stat-value";
      v.textContent = String(value);
      var l = document.createElement("span");
      l.className = "fleet-stat-label";
      l.textContent = label;
      text.appendChild(v);
      text.appendChild(l);

      li.appendChild(svg);
      li.appendChild(text);
      stats.appendChild(li);
    });

    clear(features);
    (vehicle.featureHighlights || []).forEach(function (f) {
      var li = document.createElement("li");
      li.textContent = f.name;
      features.appendChild(li);
    });

    intro.textContent = vehicle.intro1 || "";
    intro.hidden = !vehicle.intro1;

    clear(tech);
    [
      ["Länge", vehicle.length != null ? vehicle.length + " cm" : null],
      ["Breite", vehicle.width != null ? vehicle.width + " cm" : null],
      ["Höhe", vehicle.height != null ? vehicle.height + " cm" : null],
      ["Zul. Gesamtmasse", vehicle.maxWeight != null ? vehicle.maxWeight + " kg" : null]
    ].forEach(function (pair) {
      if (!pair[1]) return;
      var dt = document.createElement("dt");
      dt.textContent = pair[0];
      var dd = document.createElement("dd");
      dd.textContent = pair[1];
      tech.appendChild(dt);
      tech.appendChild(dd);
    });
    techHead.hidden = tech.children.length === 0;

    setupCalendar(vehicle);

    price.textContent = vehicle.pricePerNightFrom != null
      ? "ab " + vehicle.pricePerNightFrom + " € / Nacht"
      : "";

    var stationId = vehicle["station.id"];
    var bookURL = "https://wl.be.rentandtravel.de/detail?articleId=" + encodeURIComponent(vehicle.id)
      + (stationId != null ? "&station=" + encodeURIComponent(stationId) : "")
      + "&locale=de-DE&vehicleType=motorhome";
    book.href = bookURL;

    dialog.showModal();
  }

  document.querySelectorAll(".fleet-card-link").forEach(function (btn) {
    btn.addEventListener("click", function () {
      try {
        openDialog(JSON.parse(btn.getAttribute("data-vehicle")));
      } catch (e) {
        // Malformed data for this item — don't break the page (the card
        // itself already shows the key facts), but don't hide the error
        // either; a site owner should see this in the console.
        console.error("fleet dialog: failed to open", e);
      }
    });
  });

  dialog.addEventListener("click", function (e) {
    // A click that lands on the <dialog> element itself (not a descendant)
    // is a click on the ::backdrop area — close on backdrop click.
    if (e.target === dialog) dialog.close();
  });
  var closeBtn = dialog.querySelector("[data-fleet-close]");
  if (closeBtn) closeBtn.addEventListener("click", function () { dialog.close(); });
})();
