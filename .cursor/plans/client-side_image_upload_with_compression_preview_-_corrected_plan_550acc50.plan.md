---
name: Client-Side Image Upload with Compression Preview - Corrected Plan
overview: Implement client-side image upload with preset selection in footer, and before/after image comparison slider in main area. Fix UI freeze by moving compression to Web Worker.
todos:
  - id: create-web-worker
    content: Create Web Worker (worker.ts) for off-main-thread compression
    status: pending
  - id: update-app-worker-integration
    content: Update App.res to use worker instead of direct WASM calls
    status: pending
  - id: enhance-compare-slider
    content: Enhance CompareView slider with better visuals and performance
    status: pending
  - id: add-size-formatting
    content: Add size formatting utility and update CompareView/FileQueue badges
    status: pending
  - id: add-progress-indicators
    content: Add progress/loading states in FileQueue and CompareView
    status: pending
  - id: test-freeze-fix
    content: Test UI responsiveness during compression
    status: pending
  - id: accessibility-review
    content: Review and fix accessibility issues
    status: pending
---

## Overview

This plan implements a complete client-side image upload experience for go-pixo with:

1. **Preset selector in footer (BottomBar)** - User selects compression quality BEFORE upload using a slider
2. **Before/After image comparison slider (CompareView)** - After upload, shows original vs compressed images with draggable slider
3. **Size badges** - Show original and compressed file sizes with savings percentage
4. **Fix UI freeze** - Move compression to Web Worker to prevent blocking main thread

## User Stories & Acceptance Criteria

### User Story 1: Preset Selection in Footer (Before Upload)

**As a** user preparing to upload images

**I want to** select compression preset in the footer using a slider

**So that** I can choose the balance between file size and quality before processing

**Acceptance Criteria:**

- Footer contains a slider with 3 preset options:
  - Left label: "Smaller" (maximum compression, ~70% smaller)
  - Center label: "Balanced" (moderate compression, ~50% smaller)
  - Right label: "Best Quality" (minimal compression, ~20% smaller)
- Slider is positioned to the left of the Lossless toggle
- User can drag slider to select preset before or after upload
- Changing preset re-compresses the currently selected image
- Visual feedback shows which preset is active
- Slider is accessible via keyboard (Arrow keys to move, Enter to confirm)

**Files to modify:**

- `web/src/components/BottomBar.res` - Keep existing slider, enhance labels
- `web/src/App.res` - Ensure preset changes trigger re-compression

---

### User Story 2: Before/After Image Comparison Slider (After Upload)

**As a** user who has compressed an image

**I want to** drag a slider to compare original and compressed versions

**So that** I can visually assess the quality difference

**Acceptance Criteria:**

- After upload and compression, CompareView displays both images
- Draggable vertical slider allows user to reveal/hide original vs compressed
- Original image shown on left side of slider position
- Compressed image shown on right side of slider position
- Slider handle is clearly visible with visual indicator (arrows or handle)
- Both images maintain aspect ratio and fit within container (max height ~500px)
- Slider can be toggled between "Slider View" and "Side-by-Side" view via button
- Movement is smooth (60fps), no jank during drag
- User can drag slider left/right to compare at any position

**Files to modify:**

- `web/src/components/CompareView.res` - Enhance existing slider implementation for better UX
- `web/src/App.res` - Pass image data to CompareView

---

### User Story 3: Size Display with Savings Percentage

**As a** user compressing images

**I want to** see file sizes before and after with savings percentage

**So that** I can evaluate compression effectiveness

**Acceptance Criteria:**

- Display original file size (e.g., "1.2 MB")
- Display compressed file size (e.g., "567 KB")
- Show savings percentage with color coding:
  - Green: significant savings (>30%)
  - Yellow: moderate savings (10-30%)
  - Gray: minimal savings (<10%)
- Size badges appear on each image in comparison view
- Size format scales appropriately (bytes → KB → MB)
- Show "Savings: 53%" or similar text

**Files to modify:**

- `web/src/components/CompareView.res` - Add formatted size badges
- `web/src/components/FileQueue.res` - Show size savings in file list
- `web/src/App.res` - Calculate and pass size data

---

### User Story 4: Fix UI Freeze During Compression

**As a** user uploading multiple images

**I want the** UI to remain responsive during compression

**So that** I can continue interacting with the application

**Acceptance Criteria:**

- UI does not freeze when processing images
- Show progress indicator: "Compressing..." with spinner or animation
- User can:
  - Scroll the page
  - Select different files from queue
  - Interact with preset slider
  - Switch between compare modes
- Processing happens in background Web Worker
- Memory usage remains stable (no leaks from worker messages)
- Multiple files can queue without blocking

**Files to create:**

- `web/src/worker.ts` - Web Worker for off-main-thread compression

**Files to modify:**

- `web/src/App.res` - Replace direct WASM calls with worker communication
- `web/src/wasm.ts` - Add worker message handling
- `web/src/components/FileQueue.res` - Add progress/loading indicators

---

### User Story 5: Upload Flow

**As a** user uploading images

**I want to** upload images and immediately see comparison results

**So that** I can quickly review compression quality

**Acceptance Criteria:**

- Click "Select Files" or drag/drop images to upload
- After upload, CompareView automatically shows the image
- Image shows in comparison slider with original vs compressed
- First uploaded image is auto-selected
- File queue shows all uploaded files with status
- Progress indicator appears during compression
- After compression completes, comparison slider becomes interactive

**Files to modify:**

- `web/src/App.res` - Update upload and auto-selection logic
- `web/src/components/CompareView.res` - Show initial loading state
- `web/src/components/Dropzone.res` - No changes needed (already works)

---

## Current Architecture vs. Target Architecture

### Current Flow (Main Thread - Causes Freeze)

```
User Upload → App.res (direct WASM call) → Wasm.encodePngImageWithOptions()
              ↓ (BLOCKS MAIN THREAD)
              UI Freeze
```

### Target Flow (Web Worker - Responsive)

```
User Upload → App.res → postMessage to Worker
              ↓ (non-blocking)
              UI stays responsive
              ↓
              Worker → Wasm.encodePngImageWithOptions()
              ↓
              postMessage back to App
              ↓
              App → Update CompareView with result
```

## Implementation Order

1. **Week 1, Days 1-2:** Create Web Worker infrastructure and fix freeze issue
2. **Week 1, Days 3-4:** Enhance CompareView slider and size badges
3. **Week 1, Days 5:** Add progress indicators and polish
4. **Week 2, Days 1-2:** Testing, accessibility, and bug fixes

## Key Files Reference

```
web/
├── src/
│   ├── App.res              # Main app with reducer and upload logic
│   ├── worker.ts            # NEW - Web Worker for compression
│   ├── wasm.ts              # WASM bridge utilities
│   ├── types.res            # Type definitions
│   └── components/
│       ├── BottomBar.res    # Preset slider and controls
│       ├── CompareView.res  # Image comparison slider
│       ├── Dropzone.res     # File upload area
│       └── FileQueue.res    # File list with status
```

## Success Metrics

- UI remains responsive when processing 5+ images
- Compression presets complete in <3 seconds for typical images
- Comparison slider moves smoothly (60fps)
- 100% keyboard accessibility
- No memory leaks during batch processing