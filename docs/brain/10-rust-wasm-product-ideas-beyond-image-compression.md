# 10 Rust + WASM product ideas (beyond image compression)

If you can build **image compression in Rust and run it in the browser via WASM**, you can build many other “local file tools” that people can use without uploading data to a server.

This doc is meant to complement `docs/introduction-to-image-compression.md` by showing **what else fits the same Rust→WASM pattern**.

> If you **don’t want Rust**, you can still build “web tools” with **Go + WASM** (details near the end).  
> Just note: this particular repo is Rust, so switching to Go would mean a separate project or a rewrite.

---

## Can this be done without a server?

Yes. If your product is a “local file tool”, it can be fully client-side:

- users select files via the browser (`File` API)
- your app reads bytes into memory (`arrayBuffer()`)
- WASM code transforms bytes (compress/convert/analyze)
- the app downloads results (`Blob` + `URL.createObjectURL`)

No upload and no API is required.

Main limitations to keep in mind:

- Browser memory/CPU limits (large files can crash/slow the tab)
- No direct filesystem access (only what the user selects, unless you use newer APIs)
- Some formats are big/complex (PDF) → still possible, but engineering-heavy

---

## What kinds of products work well with Rust + WASM?

Best matches:

- **Byte-in / byte-out** tasks (take a file as bytes, produce a new file as bytes)
- **CPU-heavy** work where Rust helps performance
- **Privacy-friendly** tools (everything runs locally in the browser)
- **Deterministic** transformations (easy to test with “golden files”)

Harder matches:

- Anything that requires OS access (full filesystem, arbitrary networking) from inside WASM
- Very large formats where “from scratch” is huge (PDF is one of these)

Typical architecture looks like this:

1. Web UI: user selects a file (`File` API)
2. JS reads bytes (`arrayBuffer()`)
3. JS calls `wasm_bindgen` exported functions (usually with `Uint8Array`)
4. Rust transforms bytes and returns new `Vec<u8>`
5. JS creates a `Blob` + download link

---

## 10 ideas (simple for users, real engineering for you)

Each idea includes: **User value**, **Why WASM fits**, and **Engineering work**.

### 1) PDF Split / Merge / Reorder / Rotate

- User value: common “I just need to split 1–3 pages” workflow.
- Why WASM fits: privacy (documents never leave the device).
- Engineering work: PDF parsing is complex; recommended path is “use an existing Rust PDF library first”, then later decide if you want deeper control.

### 2) PDF “Optimizer” (shrink PDFs by recompressing embedded images)

- User value: “email size limit” problem.
- Why WASM fits: can reuse your image pipeline knowledge; also privacy.
- Engineering work: parse PDF objects → locate embedded images → decode → recompress (PNG/JPEG) → rewrite PDF xref/streams. This is more advanced than split/merge.

### 3) ZIP Builder / Extractor (folders ↔ `.zip`)

- User value: send a folder as one file, or extract without installing apps.
- Why WASM fits: pure byte work; compression is CPU-heavy.
- Engineering work: implement ZIP container format + DEFLATE (or reuse your DEFLATE code), file metadata, directory entries, streaming/large file handling, progress UI.

### 4) “Web Gzip/Brotli” Compressor for Text Assets

- User value: compress large `.json/.csv/.txt` or web assets before upload.
- Why WASM fits: fast compression; works offline.
- Engineering work: implement compressor + decompressor, verify against known test vectors, expose presets (fast/balanced/max), and handle UTF-8 vs raw bytes correctly.

### 5) Offline Encryption Tool (encrypt/decrypt files locally)

- User value: “lock this file with a password” without trusting a server.
- Why WASM fits: cryptography in Rust can be fast and consistent; browser storage optional.
- Engineering work: key derivation (Argon2/scrypt/PBKDF2), AEAD (AES-GCM / ChaCha20-Poly1305), file header format, streaming for large files, safe memory handling.

### 6) JSON/CSV Cleaner + Validator (schema checks + fixes)

- User value: “my CSV is messy, fix it” or “validate this JSON before I import”.
- Why WASM fits: pure parsing + transformation; can run on sensitive datasets locally.
- Engineering work: robust parsers, clear error messages with byte/line positions, large-file performance, incremental parsing, and deterministic output formatting.

### 7) Image/Document Metadata Stripper (privacy tool)

- User value: remove EXIF/GPS from photos, remove PDF metadata, remove author/tool info.
- Why WASM fits: privacy-focused and fast.
- Engineering work: format-specific metadata parsing, careful rewriting so files remain valid, and a UI that clearly shows “before vs after metadata”.

### 8) SVG Optimizer (minify + simplify vector files)

- User value: smaller SVGs for websites and UI.
- Why WASM fits: parse + rewrite XML; can be CPU-heavy for big SVGs.
- Engineering work: XML parsing, path simplification, removing unused defs, consistent formatting, and correctness testing (SVG render output should not change).

### 9) Audio “Utility Belt” (trim, normalize, fade, resample)

- User value: simple edits without a DAW.
- Why WASM fits: DSP math runs well in WASM; privacy.
- Engineering work: decode/encode decisions (start with WAV first—it’s simplest), resampling, peak/RMS measurement, clipping prevention, and handling long audio efficiently.

### 10) Hashing + Integrity Tool (SHA-256 / file fingerprints)

- User value: verify downloads, compare files, deduplicate.
- Why WASM fits: fast hashing for large files; runs offline.
- Engineering work: streaming hashing (chunked reads), progress UI, multi-file concurrency, and consistent output formats.

---

## Recommendation: start with a “byte tool” MVP

If your goal is “easy for people to use”, the best first products usually look like:

- input file(s) → output file(s)
- no account/login
- no server required
- immediate download

Good beginner→intermediate starting points from the list:

- **ZIP builder/extractor** (real engineering, clear UX)
- **Metadata stripper** (privacy story, narrow scope)
- **JSON/CSV validator/cleaner** (useful and testable)

---

## If you prefer Go (Golang) instead of Rust

Yes, Go can run in the browser via WASM too, but it’s **not a drop-in replacement** for Rust’s `wasm-bindgen` style.

You have 3 realistic options:

### Option A: Standard Go → WASM (official toolchain)

- Build: `GOOS=js GOARCH=wasm go build -o main.wasm ./...`
- You must also ship Go’s runtime JS helper file `wasm_exec.js` (path depends on Go version, commonly under `$(go env GOROOT)/misc/wasm/` or `$(go env GOROOT)/lib/wasm/`).
- How calls work: you usually expose functions to JS via `syscall/js` (set a global function, or call back into JS). JS can’t “directly import Go functions” the same way as `wasm-bindgen`; Go runs as a runtime that you interact with through `syscall/js`.

This is a good fit for:
- simple utilities, hashing, transforms, parsers,
- when you already know Go and accept a larger runtime.

Common trade-offs:
- larger `.wasm` output than Rust for small tools,
- data passing often involves copying between JS and Go memory,
- you need to manage “start the Go runtime” in the browser before calling functions.

### Option B: TinyGo → WASM (smaller binaries)

TinyGo compiles a subset of Go and often produces **much smaller WASM**.
It’s a strong choice for “download a tool and run locally” products where size matters.

Trade-offs:
- not full Go compatibility (some stdlib/features differ),
- you must design APIs carefully to keep JS↔WASM data movement efficient.

### Option C: Go backend + web UI (no WASM)

If your tool is huge (PDF is a common example), a practical MVP is:

- Web UI (drag/drop)
- Go server API does the heavy processing

Trade-offs:
- users must upload files (privacy concerns),
- you must run infrastructure,
- but engineering can be simpler than full in-browser WASM for complex formats.

---

## One question (pick one)

Which category do you want your first product to be?

1) **Documents** (PDF tools), 2) **Compression/archives** (zip/gzip), 3) **Privacy/security** (encrypt/strip metadata), 4) **Data tools** (CSV/JSON), 5) **Audio**.

---

## 30 more “no server” tool ideas (Go/JS/TS + WASM)

All of these can be “easy for people” (upload/select → click → download), but still teach real engineering.

### Documents & Office-ish (not only PDF)

1. PDF page **extract** (select pages → new PDF)
2. PDF **rotate** pages (fix scanned sideways docs)
3. PDF **watermark** (text stamp)
4. PDF **remove password** (if user knows it)
5. PDF **add password** (encrypt)
6. PDF **metadata viewer/stripper** (author, tool, dates)
7. DOCX → **plain text** extractor (offline)
8. PPTX → **export slide images** (png/jpeg)
9. E-book **EPUB inspector** (list resources, extract text)
10. “**Print to PDF**” post-processor (shrink/clean common outputs)

### Archives & packaging

11. ZIP **create** (many files → zip)
12. ZIP **extract** (zip → files)
13. TAR **pack/unpack**
14. Gzip **compress/decompress** any file
15. “**Rezip**” optimizer (rewrite zip with better compression settings)

### Media (beyond images)

16. WAV **trim** (cut start/end) + export WAV
17. WAV **normalize** volume (peak/RMS) + export WAV
18. WAV **fade in/out** + export WAV
19. Simple audio **resampler** (48k → 44.1k)
20. MP3/MP4 **metadata viewer** (ID3 / container tags) and stripper (where feasible)

### Privacy & security

21. File **hashing** tool (SHA-256, SHA-1, MD5) for integrity checks
22. Folder/file **duplicate finder** (hash + compare)
23. Local **encrypt/decrypt** (password-based) with a small custom file format
24. **Secret scanner** for text files (find API keys/tokens before committing)
25. “**Safe share**” bundle: strip metadata + zip + encrypt in one click

### Developer/data tools

26. JSON **formatter/minifier** + JSONPath query runner
27. CSV **cleaner** (delimiter detect, trim, normalize quotes) + export CSV
28. “CSV → JSON” and “JSON → CSV” converter (with schema hints)
29. Log **redactor** (mask emails/phones/UUIDs) + export sanitized text
30. “**Diff two files**” (binary/text) + generate patch/summary report

If you tell me your top 3 from this list, I can propose a clean MVP scope and a WASM-friendly API shape (inputs/outputs, chunking, progress UI) for whichever language you pick (Go or Rust).
