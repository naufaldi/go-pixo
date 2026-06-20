# AI and SEO Discoverability Guide

How Go-Pixo exposes public metadata for search engines and AI systems, and how to extend it safely on future projects.

## Why this exists

Search engines and AI crawlers need more than a React app shell. They rely on:

- HTML metadata and structured data
- Static files at predictable URLs (`/robots.txt`, `/sitemap.xml`, `/llms.txt`)
- Consistent product copy across HTML, UI, and machine-readable files

Go-Pixo is a Vite single-page app. We use **static public files** instead of a Next.js metadata layer because the site has one public route.

## File roles (do not mix them up)

| File | Purpose | Blocks crawlers? |
|------|---------|------------------|
| `robots.txt` | Access control for crawlers | Yes (by convention) |
| `sitemap.xml` | URL discovery for search engines | No |
| `llms.txt` | Short curated index for LLMs | No |
| `llms-full.txt` | Fuller grounded context for LLMs | No |
| `.well-known/ai.txt` | Explicit AI usage and citation policy | No |
| `index.html` meta + JSON-LD | Page title, social previews, rich results | No |
| `site.webmanifest` | PWA identity, icons, shortcuts | No |

**Rule:** `robots.txt` controls *permission*. `llms.txt` controls *curation*. They solve different problems and should both exist.

## Go-Pixo layout

```
web/
├── index.html              # title, OG/Twitter, JSON-LD, noscript fallback, llms discovery link
└── public/
    ├── robots.txt
    ├── sitemap.xml
    ├── llms.txt
    ├── llms-full.txt
    ├── site.webmanifest
    ├── .well-known/ai.txt
    └── og-image.png, favicon.*, icon-*.png
```

Vite copies `web/public/` into `web/dist/` on build. Caddy serves `dist/` in production.

Canonical site URL: `https://go-pixo.faldi.xyz/`

## Checklist for a new site or feature

### 1. Baseline SEO (`index.html`)

- [ ] `<title>` and `<meta name="description">`
- [ ] `<link rel="canonical">` with production URL
- [ ] `<meta name="robots" content="index, follow, max-image-preview:large">`
- [ ] Open Graph tags (`og:title`, `og:description`, `og:image`, `og:url`, `og:image:alt`)
- [ ] Twitter Card tags (`twitter:card`, `twitter:title`, `twitter:description`, `twitter:image`, `twitter:image:alt`)
- [ ] JSON-LD (`WebApplication`, `Organization`, or `WebSite` as appropriate)
- [ ] Favicon + manifest links
- [ ] `<noscript>` fallback with core value proposition (SPAs often render nothing without JS)

### 2. Crawler files (`web/public/`)

- [ ] `robots.txt` — allow public paths; disallow private paths (e.g. `/test-fixtures/`, `/api/`)
- [ ] Explicit allow rules for major crawlers if you want AI/search visibility: `Googlebot`, `Bingbot`, `GPTBot`, `ClaudeBot`, `PerplexityBot`, `Google-Extended`, `CCBot`
- [ ] `Sitemap:` line pointing to production sitemap URL
- [ ] `sitemap.xml` — list every public URL with `loc`, `lastmod`, `changefreq`, `priority`
- [ ] `site.webmanifest` — `name`, `icons`, `theme_color`, optional `categories` and `shortcuts`

### 3. AI / LLM files

- [ ] `llms.txt` — short index: H1, blockquote summary, scope notes, canonical links
- [ ] `llms-full.txt` — detailed public context: overview, user journey, supported features, privacy limits, non-goals
- [ ] `.well-known/ai.txt` — AI permissions, attribution preference, privacy scope
- [ ] `<link rel="alternate" href="/llms.txt">` in `index.html` for discovery
- [ ] Cross-link files: `llms.txt` → `llms-full.txt` → `ai.txt` → `robots.txt` → `sitemap.xml`

### 4. Copy consistency

- [ ] Hero/UI text matches meta description (e.g. PNG **and** JPEG, not PNG only)
- [ ] `llms.txt` and `llms-full.txt` only claim features present in the public UI
- [ ] Distinguish **public site content** from **user-uploaded local data** (important for Go-Pixo)

### 5. Verify before merge

```bash
cd web && bun run build
python3 -m json.tool public/site.webmanifest
python3 -c "import xml.etree.ElementTree as ET; ET.parse('public/sitemap.xml')"
ls dist/llms.txt dist/llms-full.txt dist/.well-known/ai.txt dist/robots.txt dist/sitemap.xml
```

After deploy, verify live:

- `https://<domain>/robots.txt`
- `https://<domain>/sitemap.xml`
- `https://<domain>/llms.txt`
- `https://<domain>/llms-full.txt`
- `https://<domain>/.well-known/ai.txt`

Submit the sitemap in Google Search Console.

## Lessons from adapting `sol-homepage`

Reference: Sukanda OneLink landing (`sol-homepage` on GitLab). Strong patterns we adopted:

1. **Treat discoverability as a public contract** — not only meta tags in the HTML shell.
2. **Two-tier LLM context** — `llms.txt` (index) + `llms-full.txt` (full summary).
3. **Explicit AI policy** — `.well-known/ai.txt` states retrieval, indexing, summarization, embedding, and training scope.
4. **Central site facts** — in Next.js they use `lib/seo.ts`; in Go-Pixo we keep canonical URLs consistent across static files manually (acceptable for a single-page app).
5. **Curated sitemap** — multi-route apps generate sitemaps from public routes; Go-Pixo keeps a single homepage entry with `lastmod`.
6. **PWA manifest polish** — `id`, `categories`, `orientation`, shortcuts improve install/discovery surfaces.

Patterns we **did not** port (not needed for Go-Pixo today):

- Next.js `metadata` API and per-route `generateMetadata`
- Dynamic sitemap from API-backed routes
- `keywords`, Google site verification (add when you have accounts/keys)
- SSR or prerender (optional future improvement for SPA body content)

## SPA-specific gotchas

1. **Body content is JS-rendered** — crawlers may not see React-only H1/tagline. Mitigate with `index.html` meta, JSON-LD, `<noscript>`, and static LLM files.
2. **Do not edit `web/lib/`** — ReScript source lives in `web/src/`; `web/lib/` is generated output.
3. **Exclude non-public assets** — disallow `/test-fixtures/` in `robots.txt`.
4. **Deploy is required** — files in git are not live until built and deployed; always check production URLs after merge.

## When to extend

| Change | Update |
|--------|--------|
| New public page/route | Add URL to `sitemap.xml`; add section to `llms-full.txt` |
| Product positioning change | Sync `index.html`, `App.res` hero, `llms.txt`, `llms-full.txt` |
| New AI crawler | Add `User-agent` allow rule in `robots.txt` |
| Stricter AI policy | Update `.well-known/ai.txt` and cross-links in `llms.txt` |
| Rebrand / domain change | Update canonical URL in all files (grep for `go-pixo.faldi.xyz`) |

## Related issues

This guide supports the SEO/discoverability work tracked in GitHub issues #17–#20 and implemented in PR #21.
