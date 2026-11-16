/* Landing page (GSAP 3 + ScrollTrigger).
 * Navigation, copy buttons and terminal replay are shared with my other
 * project sites; the request pipeline is specific to this one. Without
 * JavaScript all content is visible; with prefers-reduced-motion the
 * diagram shows its final state. */
(function () {
  "use strict";
  var root = document.getElementById("lp");
  if (!root) return;
  var gsap = window.gsap;
  if (gsap && window.ScrollTrigger) gsap.registerPlugin(window.ScrollTrigger);
  var reduce = window.matchMedia && window.matchMedia("(prefers-reduced-motion: reduce)").matches;
  var animate = !!gsap && !reduce;

  function whenVisible(el, fn, margin) {
    if (!("IntersectionObserver" in window)) { fn(); return; }
    var io = new IntersectionObserver(function (entries) {
      entries.forEach(function (e) { if (e.isIntersecting) { io.disconnect(); fn(); } });
    }, { rootMargin: margin || "0px 0px -10% 0px" });
    io.observe(el);
  }

  /* ---------------- navigation: solid header and section dots ---------------- */
  (function nav() {
    var bar = document.getElementById("lp-nav");
    var onScroll = function () { if (bar) bar.classList.toggle("is-solid", window.scrollY > 24); };
    window.addEventListener("scroll", onScroll, { passive: true });
    onScroll();
    var dots = Array.prototype.slice.call(document.querySelectorAll(".lp-dots a"));
    if (!dots.length || !("IntersectionObserver" in window)) return;
    var io = new IntersectionObserver(function (entries) {
      entries.forEach(function (e) {
        if (!e.isIntersecting) return;
        dots.forEach(function (d) { d.classList.toggle("is-on", d.getAttribute("data-dot") === e.target.id); });
      });
    }, { rootMargin: "-45% 0px -50% 0px" });
    dots.forEach(function (d) {
      var sec = document.getElementById(d.getAttribute("data-dot"));
      if (sec) io.observe(sec);
    });
  })();

  /* ---------------- copy buttons ---------------- */
  root.querySelectorAll("[data-copy], [data-copy-target]").forEach(function (btn) {
    btn.addEventListener("click", function () {
      var text = btn.getAttribute("data-copy");
      if (!text) {
        var t = document.getElementById(btn.getAttribute("data-copy-target"));
        text = t ? t.textContent : "";
      }
      if (!navigator.clipboard) return;
      navigator.clipboard.writeText(text.trim()).then(function () {
        var label = btn.textContent;
        btn.textContent = "Copied";
        setTimeout(function () { btn.textContent = label; }, 1500);
      });
    });
  });

  /* ---------------- terminal replay ----------------
   * <pre class="lp-term"> holds one <span class="ln"> per line (visible
   * without JS). When it scrolls into view the lines are replayed. */
  root.querySelectorAll("pre.lp-term").forEach(function (term) {
    if (reduce) return;
    var lines = Array.prototype.slice.call(term.querySelectorAll(".ln"));
    if (!lines.length) return;
    lines.forEach(function (l) { l.style.display = "none"; });
    var cur = document.createElement("span");
    cur.className = "cur";
    cur.textContent = "▋";
    term.appendChild(cur);
    whenVisible(term, function () {
      var i = 0;
      (function next() {
        if (i >= lines.length) { cur.remove(); return; }
        var l = lines[i++];
        l.style.display = "";
        term.insertBefore(cur, l.nextSibling);
        setTimeout(next, l.classList.contains("cmd") ? 650 : 170);
      })();
    });
  });

  /* ---------------- tabs ----------------
   * Without JavaScript every panel is visible; the "hidden" attribute only
   * takes effect once the container is marked ready. */
  root.querySelectorAll(".lp-tabs").forEach(function (tabs) {
    var buttons = Array.prototype.slice.call(tabs.querySelectorAll("[data-tab]"));
    var panels = Array.prototype.slice.call(tabs.querySelectorAll("[data-panel]"));
    function select(name) {
      buttons.forEach(function (b) { b.setAttribute("aria-selected", b.getAttribute("data-tab") === name ? "true" : "false"); });
      panels.forEach(function (p) { p.hidden = p.getAttribute("data-panel") !== name; });
    }
    buttons.forEach(function (b) { b.addEventListener("click", function () { select(b.getAttribute("data-tab")); }); });
    tabs.classList.add("is-ready");
    if (buttons.length) select(buttons[0].getAttribute("data-tab"));
  });

  /* ---------------- request pipeline ----------------
   * The stages are <g data-stage="n" data-cx data-cy> in document order.
   * Each step button names the range of stages it covers (data-from,
   * data-to); the step marked data-back walks the response back out. */
  (function pipeline() {
    var svg = root.querySelector('.lp-diagram[data-flow="pipeline"]');
    if (!svg) return;
    var stages = Array.prototype.slice.call(svg.querySelectorAll("[data-stage]"));
    var packet = svg.querySelector("#lp-p-req");
    var status = svg.querySelector("#lp-status");
    var buttons = Array.prototype.slice.call(root.querySelectorAll("[data-step]"));
    var caption = document.getElementById("lp-flow-caption");
    var toggle = root.querySelector('[data-flow="toggle"]');
    var restart = root.querySelector('[data-flow="restart"]');
    if (!stages.length || !packet) return;

    function centre(stage) {
      return { cx: parseFloat(stage.getAttribute("data-cx")), cy: parseFloat(stage.getAttribute("data-cy")) };
    }
    function setStep(i) {
      buttons.forEach(function (b, j) {
        if (j === i) b.setAttribute("aria-current", "step"); else b.removeAttribute("aria-current");
      });
      var b = buttons[i];
      if (!b) return;
      if (status) status.textContent = b.getAttribute("data-status") || "";
      if (caption) caption.textContent = b.getAttribute("data-caption") || "";
    }
    function clear() {
      stages.forEach(function (s) { s.classList.remove("is-seen", "is-focus", "is-ok"); });
    }

    if (!animate) {
      stages.forEach(function (s) { s.classList.add("is-ok"); });
      setStep(buttons.length - 1);
      buttons.forEach(function (b, i) { b.addEventListener("click", function () { setStep(i); }); });
      if (toggle) toggle.hidden = true;
      if (restart) restart.hidden = true;
      return;
    }

    var labels = buttons.map(function (_, i) { return "s" + i; });
    var tl = gsap.timeline({ repeat: -1, repeatDelay: 1.4, paused: true });
    tl.call(clear);
    tl.set(packet, { opacity: 0, attr: centre(stages[0]) });

    buttons.forEach(function (button, i) {
      tl.addLabel(labels[i]);
      tl.call(setStep, [i]);

      if (button.hasAttribute("data-back")) {
        // The response: every stage it leaves turns green.
        tl.call(function () { packet.classList.add("lp-d-pkt--ok"); packet.classList.remove("lp-d-pkt--req"); });
        tl.set(packet, { opacity: 1, attr: centre(stages[stages.length - 1]) });
        for (var k = stages.length - 1; k >= 0; k--) {
          (function (stage) {
            tl.to(packet, { attr: centre(stage), duration: 0.13, ease: "none" });
            tl.call(function () { stage.classList.remove("is-seen", "is-focus"); stage.classList.add("is-ok"); });
          })(stages[k]);
        }
        tl.to(packet, { opacity: 0, duration: 0.25 });
        tl.to({}, { duration: 1.6 });
        return;
      }

      var from = parseInt(button.getAttribute("data-from"), 10);
      var to = parseInt(button.getAttribute("data-to"), 10);
      // Jumping to a step shows the earlier stages as already crossed.
      tl.call(function () {
        packet.classList.add("lp-d-pkt--req"); packet.classList.remove("lp-d-pkt--ok");
        stages.forEach(function (s, n) {
          s.classList.remove("is-focus", "is-ok");
          s.classList.toggle("is-seen", n < from);
        });
      });
      tl.set(packet, { opacity: 1, attr: centre(stages[Math.max(from - 1, 0)]) });
      for (var n = from; n <= to; n++) {
        (function (stage, previous) {
          tl.to(packet, { attr: centre(stage), duration: 0.3, ease: "power1.inOut" });
          tl.call(function () {
            if (previous) { previous.classList.remove("is-focus"); previous.classList.add("is-seen"); }
            stage.classList.add("is-focus");
          });
          tl.to({}, { duration: 0.16 });
        })(stages[n], stages[n - 1]);
      }
      tl.to({}, { duration: 0.7 });
    });

    function setPlaying(p) {
      if (!toggle) return;
      toggle.setAttribute("aria-label", p ? "Pause animation" : "Play animation");
      toggle.textContent = p ? "❚❚" : "▶";
      toggle.dataset.paused = p ? "" : "1";
    }
    buttons.forEach(function (b, i) { b.addEventListener("click", function () { tl.play(labels[i]); setPlaying(true); }); });
    if (toggle) toggle.addEventListener("click", function () {
      if (tl.paused()) { tl.play(); setPlaying(true); } else { tl.pause(); setPlaying(false); }
    });
    if (restart) restart.addEventListener("click", function () { tl.restart(); setPlaying(true); });

    var allowed = function () { return !toggle || !toggle.dataset.paused; };
    if (window.ScrollTrigger) {
      window.ScrollTrigger.create({
        trigger: svg, start: "top 90%", end: "bottom 10%",
        onEnter: function () { if (allowed()) tl.play(); },
        onEnterBack: function () { if (allowed()) tl.play(); },
        onLeave: function () { tl.pause(); },
        onLeaveBack: function () { tl.pause(); }
      });
    } else {
      tl.play();
    }
  })();
})();
