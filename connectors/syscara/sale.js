// Syscara sales detail dialog — gallery, key facts, equipment.
//
// Ships as an external file, not inline in sale.tmpl.html: webhull's
// default CSP has no 'unsafe-inline' and no nonce for script-src, so an
// inline <script> would be silently dropped. Copy this file to the
// consuming site's static directory as /static/js/syscara-sale.js.
(function () {
  "use strict";
  var dialog = document.getElementById("sale-detail-dialog");
  if (!dialog || typeof dialog.showModal !== "function") return;

  var img = document.getElementById("sale-dialog-img");
  var thumbs = document.getElementById("sale-dialog-thumbs");
  var title = document.getElementById("sale-dialog-title");
  var group = document.getElementById("sale-dialog-group");
  var price = document.getElementById("sale-dialog-price");
  var inquire = document.getElementById("sale-dialog-inquire");
  var stats = document.getElementById("sale-dialog-stats");
  var features = document.getElementById("sale-dialog-features");
  var techHead = document.getElementById("sale-dialog-tech-head");
  var tech = document.getElementById("sale-dialog-tech");
  var equipmentHead = document.getElementById("sale-dialog-equipment-head");
  var equipment = document.getElementById("sale-dialog-equipment");
  var inquireBase = inquire ? inquire.getAttribute("href") : "";

  // Syscara's vocabulary for features, beds, gearboxes and fuels. Unknown
  // keys fall back to a readable form of the key itself.
  var FEATURES = {
    abs: "ABS", allwetterreifen: "Allwetterreifen", android_auto: "Android Auto",
    anhaengerkupplung_fest: "Anhängerkupplung", apple_carplay: "Apple CarPlay",
    autark: "Autark", esp: "ESP", heckgarage: "Heckgarage", isofix: "Isofix",
    klimaanlage: "Klimaanlage", kompressor: "Kompressor-Kühlschrank", lithium: "Lithium-Batterie",
    luftfederung: "Luftfederung", markise: "Markise", multifunktionslenkrad: "Multifunktionslenkrad",
    rueckfahrkamera: "Rückfahrkamera", sat: "Sat-Anlage", sep_dusche: "Separate Dusche",
    solar: "Solaranlage", tempomat: "Tempomat", wc: "WC", winterreifen: "Winterreifen"
  };
  var BEDS = {
    FRENCH_BED: "Französisches Bett", DOUBLE_BED: "Doppelbett", SINGLE_BEDS: "Einzelbetten",
    QUEEN_BED: "Queensbett", PULL_BED: "Hubbett", ROOF_BED: "Dachbett", ALCOVE_BED: "Alkovenbett",
    BUNK_BED: "Stockbett", DINETTE_BED: "Sitzgruppenbett", TRANSVERSE_BED: "Querbett"
  };
  var GEAR = { AUTOMATIC: "Automatik", MANUAL: "Schaltgetriebe" };
  var FUEL = { DIESEL: "Diesel", PETROL: "Benzin", GASOLINE: "Benzin", ELECTRIC: "Elektro", HYBRID: "Hybrid", LPG: "Autogas" };

  function label(map, key) {
    if (key == null || key === "") return "";
    if (map[key]) return map[key];
    var s = String(key).toLowerCase().replace(/_/g, " ");
    return s.charAt(0).toUpperCase() + s.slice(1);
  }
  function num(v) { return Number(v).toLocaleString("de-DE"); }
  function clear(el) { while (el.firstChild) el.removeChild(el.firstChild); }
  function sized(url, size) { return String(url).replace(/\/1500\.(\w+)$/, "/" + size + ".$1"); }

  function titleOf(sale) {
    return [sale["model.producer"], sale["model.series"], sale["model.model"]].filter(Boolean).join(" ");
  }

  // Photos first, floor plans ("layout") last — matched to the media list
  // by id, so a media id the API couldn't resolve doesn't shift the rest.
  function gallery(sale) {
    var groupById = {};
    (sale.media || []).forEach(function (m) { groupById[m.id] = m.group; });
    var photos = [], layouts = [];
    (sale.images || []).forEach(function (im) {
      if (!im.file) return;
      (groupById[im.id] === "layout" ? layouts : photos).push(im.file);
    });
    return photos.concat(layouts);
  }

  // texts.description is the dealer's equipment list as <ul><li> HTML.
  // Parsed inertly and read as text only — never inserted as markup.
  function equipmentItems(html) {
    if (!html) return [];
    var doc = new DOMParser().parseFromString(html, "text/html");
    var items = Array.prototype.map.call(doc.querySelectorAll("li"), function (li) {
      return li.textContent.trim();
    }).filter(Boolean);
    if (items.length === 0 && doc.body.textContent.trim()) items = [doc.body.textContent.trim()];
    return items;
  }

  function showImage(url) { img.src = sized(url, 1500); }

  function openDialog(sale) {
    var images = gallery(sale);
    var name = titleOf(sale);

    img.hidden = images.length === 0;
    if (images.length) showImage(images[0]);
    img.alt = name;

    clear(thumbs);
    images.forEach(function (url, idx) {
      var t = document.createElement("button");
      t.type = "button";
      t.className = "fleet-dialog-thumb" + (idx === 0 ? " is-active" : "");
      t.setAttribute("aria-label", "Bild " + (idx + 1) + " anzeigen");
      var ti = document.createElement("img");
      ti.src = sized(url, "sm");
      ti.alt = "";
      ti.loading = "lazy";
      t.appendChild(ti);
      t.addEventListener("click", function () {
        showImage(url);
        thumbs.querySelectorAll(".fleet-dialog-thumb").forEach(function (x) { x.classList.remove("is-active"); });
        t.classList.add("is-active");
      });
      thumbs.appendChild(t);
    });
    thumbs.hidden = images.length <= 1;

    title.textContent = name;
    group.textContent = [sale.typeof, sale["model.model_add"], sale.condition === "NEW" ? "Neufahrzeug" : ""]
      .filter(Boolean).join(" · ");

    var offer = sale["prices.offer"];
    price.textContent = offer ? num(offer) + " €" : "Preis auf Anfrage";
    if (offer && sale["prices.vat"]) {
      var note = document.createElement("span");
      note.className = "fleet-dialog-price-note";
      note.textContent = "inkl. " + sale["prices.vat"] + " % MwSt.";
      price.appendChild(document.createTextNode(" "));
      price.appendChild(note);
    }

    if (inquire) {
      var hashAt = inquireBase.indexOf("#");
      var path = hashAt < 0 ? inquireBase : inquireBase.slice(0, hashAt);
      var hash = hashAt < 0 ? "" : inquireBase.slice(hashAt);
      inquire.href = path + (path.indexOf("?") < 0 ? "?" : "&") + "fahrzeug=" + encodeURIComponent(name) + hash;
    }

    var reg = sale["date.registration"];
    var regText = reg && reg.length >= 7 ? reg.slice(5, 7) + "/" + reg.slice(0, 4) : "";
    var power = sale["engine.kw"] ? sale["engine.kw"] + " kW / " + sale["engine.ps"] + " PS" : "";

    clear(stats);
    [
      ["s-year", regText, "Erstzulassung"],
      ["s-mileage", sale.mileage != null ? num(sale.mileage) + " km" : "", "Kilometerstand"],
      ["s-power", power, "Leistung"],
      ["s-gear", label(GEAR, sale["engine.gear"]), "Getriebe"],
      ["s-beds", sale["beds.num"] || "", "Schlafplätze"],
      ["s-seats", sale.seats || "", "Sitzplätze"]
    ].forEach(function (row) {
      if (row[1] === "" || row[1] == null) return;
      var li = document.createElement("li");
      li.className = "fleet-stat";
      var svg = document.createElementNS("http://www.w3.org/2000/svg", "svg");
      svg.setAttribute("class", "fleet-stat-icon");
      svg.setAttribute("aria-hidden", "true");
      svg.setAttribute("focusable", "false");
      var use = document.createElementNS("http://www.w3.org/2000/svg", "use");
      use.setAttribute("href", "#" + row[0]);
      svg.appendChild(use);
      var text = document.createElement("span");
      text.className = "fleet-stat-text";
      var v = document.createElement("strong");
      v.className = "fleet-stat-value";
      v.textContent = String(row[1]);
      var l = document.createElement("span");
      l.className = "fleet-stat-label";
      l.textContent = row[2];
      text.appendChild(v);
      text.appendChild(l);
      li.appendChild(svg);
      li.appendChild(text);
      stats.appendChild(li);
    });

    clear(features);
    (sale.features || []).forEach(function (f) {
      var li = document.createElement("li");
      li.textContent = label(FEATURES, f);
      features.appendChild(li);
    });

    var dims = [sale["dimensions.length"], sale["dimensions.width"], sale["dimensions.height"]];
    var beds = (sale["beds.beds"] || []).map(function (b) {
      return label(BEDS, b.type) + (b.size ? " (" + b.size + " cm)" : "");
    }).join(", ");

    clear(tech);
    [
      ["Fahrzeugart", [sale.type, sale.typeof].filter(Boolean).join(", ")],
      ["Länge / Breite / Höhe", dims.every(Boolean) ? dims.join(" / ") + " cm" : ""],
      ["Zul. Gesamtmasse", sale["weights.total"] ? num(sale["weights.total"]) + " kg" : ""],
      ["Kraftstoff", label(FUEL, sale["engine.fuel"])],
      ["Modelljahr", sale["model.modelyear"] || ""],
      ["Betten", beds]
    ].forEach(function (pair) {
      if (!pair[1]) return;
      var dt = document.createElement("dt");
      dt.textContent = pair[0];
      var dd = document.createElement("dd");
      dd.textContent = String(pair[1]);
      tech.appendChild(dt);
      tech.appendChild(dd);
    });
    techHead.hidden = tech.children.length === 0;

    clear(equipment);
    equipmentItems(sale["texts.description"]).forEach(function (text) {
      var li = document.createElement("li");
      li.textContent = text;
      equipment.appendChild(li);
    });
    equipmentHead.hidden = equipment.children.length === 0;

    dialog.showModal();
  }

  document.querySelectorAll("[data-sale]").forEach(function (btn) {
    btn.addEventListener("click", function () {
      try {
        openDialog(JSON.parse(btn.getAttribute("data-sale")));
      } catch (e) {
        console.error("sale dialog: failed to open", e);
      }
    });
  });

  dialog.addEventListener("click", function (e) {
    if (e.target === dialog) dialog.close();
  });
  var closeBtn = dialog.querySelector("[data-sale-close]");
  if (closeBtn) closeBtn.addEventListener("click", function () { dialog.close(); });
})();
