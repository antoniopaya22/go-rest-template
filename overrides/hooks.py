"""MkDocs hooks of the documentation site.

- The API reference page renders api/openapi.yaml, which lives outside the
  docs directory, so the file is copied into the built site.
- The changelog and contributing pages show CHANGELOG.md and CONTRIBUTING.md
  from the repository root. Their relative links point at files of the
  repository, not at pages of the site, so the files are included here, with
  those links rewritten to GitHub URLs, before MkDocs validates links.
"""

import re
import shutil
from pathlib import Path

INCLUDED = {
    "project/changelog.md": "CHANGELOG.md",
    "project/contributing.md": "CONTRIBUTING.md",
}
PLACEHOLDER = "<!-- include: {} -->"
LINK = re.compile(r"\]\((?!https?://|mailto:|#)([^)\s]+)\)")


def _root(config):
    return Path(config["config_file_path"]).parent


def on_page_markdown(markdown, page, config, files):
    source = INCLUDED.get(page.file.src_uri)
    if source is None:
        return markdown

    placeholder = PLACEHOLDER.format(source)
    if placeholder not in markdown:
        raise ValueError(f"{page.file.src_uri} must contain the line {placeholder!r}")

    base = config["repo_url"].rstrip("/") + "/blob/main/"
    content = (_root(config) / source).read_text(encoding="utf-8")
    content = LINK.sub(lambda match: "](" + base + match.group(1).removeprefix("./") + ")", content)
    return markdown.replace(placeholder, content)


def on_post_build(config):
    shutil.copyfile(_root(config) / "api" / "openapi.yaml", Path(config["site_dir"]) / "openapi.yaml")
