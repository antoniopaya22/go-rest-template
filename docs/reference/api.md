---
hide:
  - toc
---

# HTTP API

This is `api/openapi.yaml`, the contract of the template, as it is in the repository. A
running server shows the same reference, for its own version of the document, at `/docs`.

<a class="md-button" href="../../openapi.yaml" download="openapi.yaml">Download the OpenAPI document</a>
[Open the reference in its own tab](api-viewer.html){ .md-button target="_blank" }

<iframe class="api-frame" id="api-frame" title="HTTP API reference" src="../api-viewer.html" loading="lazy"></iframe>

<script>
  // Keep the embedded reference in the colour scheme of the page.
  (function () {
    var frame = document.getElementById("api-frame");
    if (!frame) return;
    var base = frame.getAttribute("src").split("?")[0];
    function dark() { return document.body.getAttribute("data-md-color-scheme") === "slate"; }
    function apply() {
      var wanted = base + (dark() ? "?dark=1" : "?dark=0");
      if (frame.getAttribute("src") !== wanted) frame.setAttribute("src", wanted);
    }
    apply();
    new MutationObserver(apply).observe(document.body, { attributes: true, attributeFilter: ["data-md-color-scheme"] });
  })();
</script>

## Conventions

The conventions the document follows are described in the guides:

- [Errors](../architecture/errors.md): every non-2xx response is a problem document.
- [Lists, updates and validation](../guides/api-conventions.md).
- [Tokens and roles](../guides/authentication.md).
