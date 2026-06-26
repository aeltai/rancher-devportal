#!/usr/bin/env bash
# Build GitHub Pages artifact: docs site + extension bundle under extensions/
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
PKG="${1:?usage: build-pages-site.sh <krew|devportal>}"

case "$PKG" in
  krew)     EXT_DIR="extensions/krew/0.1.0/plugin"; DIST="dist-pkg/krew-0.1.0" ;;
  devportal) EXT_DIR="extensions/devportal/0.1.0/plugin"; DIST="dist-pkg/devportal-0.1.0" ;;
  *) echo "Unknown package: $PKG"; exit 1 ;;
esac

cd "$ROOT"
rm -rf _site
mkdir -p "_site/$EXT_DIR" _site/docs

cp docs/pages/index.html _site/index.html
cp docs/pages/site.css _site/site.css
cp docs/pages/index.html _site/404.html
touch _site/.nojekyll

cp -r "$DIST"/* "_site/$EXT_DIR/"

for md in docs/*.md; do
  [ -f "$md" ] || continue
  base=$(basename "$md" .md)
  [ "$base" = "README" ] && continue
  body=$(npx --yes marked "$md")
  cat > "_site/docs/${base}.html" <<EOF
<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="utf-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1" />
  <title>${base} — docs</title>
  <link rel="stylesheet" href="../site.css" />
</head>
<body>
  <main class="wrap">
    <a class="back" href="../index.html">&larr; Home</a>
    <article class="doc">
${body}
    </article>
  </main>
</body>
</html>
EOF
done

echo "Pages site ready in _site/ ($(find _site -type f | wc -l | tr -d ' ') files)"
