# SliceBridge public landing page

This directory is a small pre-release website for SliceBridge, published via GitHub Pages from `main/docs`.

- `index.html`: accessible English / Hebrew page; language preference can be selected with `?lang=en` or `?lang=he`.
- `assets/approved-icon.png` and `assets/approved-wave.png`: **exact approved branding blobs**, reused unchanged from the SliceBridge development branch.
- `assets/social-preview.png`: 1280×640 social card composed from the two approved assets; used for website Open Graph sharing. Repository Settings → Social preview still requires a separate upload through GitHub UI.
- No account, analytics, cookies, external fonts or download link.
- **Pre-release:** keep the development disclaimer visible until a signed, validated release exists.

GitHub Pages: https://nocturney.github.io/slicebridge/

Keep this site isolated from the application's UI and its open development PRs.

## Shareable media

- `press.html`: accessible bilingual media page using the original approved icon and banner.
- `assets/marketing/*.svg`: six language-specific, self-contained vector share cards (story, workflow, community). The SVGs are **supporting materials, not logo replacements**.
- Original approved banner remains unchanged. Its four corners are rounded at display time in the landing/media CSS, preserving source pixels.
- Editable/copy-ready source text lives in `../marketing/COPY.md`. No general release/download is claimed.
