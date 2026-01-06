# Documentation Package

Learning materials for image compression algorithms (PNG, JPEG, DEFLATE).

## Package Identity

- **Purpose**: Comprehensive educational resources for compression algorithms
- **Technology**: Plain Markdown files
- **Content**: Theory, implementation guides, and learning materials

## Purpose

These docs are for:
- New contributors learning the codebase
- Understanding algorithms before coding
- Architecture decisions and trade-offs
- Reference for complex implementations

## Structure

```
docs/
├── learning/
│   ├── jpg/          # JPEG documentation
│   │   ├── index.md
│   │   ├── progressive.md
│   │   ├── trellis.md
│   │   ├── huffman-optimized.md
│   │   └── options.md
│   └── png/          # PNG documentation
└── *.md              # General compression docs
```

## Key Resources

### Core Algorithms
- **PNG Spec**: `docs/learning/png/png.md` (chunk format, scanlines)
- **JPEG Spec**: `docs/learning/jpg/jpeg.md` (encoding pipeline, markers)
- **DEFLATE**: `docs/deflate.md` (compression algorithm)
- **DCT**: `docs/dct.md` (frequency transform)
- **Huffman**: `docs/huffman-coding.md` (entropy coding)

### Advanced Topics
- **Trellis**: `docs/learning/jpg/trellis.md` (rate-distortion optimization)
- **Progressive**: `docs/learning/jpg/progressive.md` (multi-scan encoding)
- **Quantization**: `docs/quantization.md` (quality control)
- **LZ77**: `docs/lz77-compression.md` (sliding window)

### Learning Path
1. **Start Here**: `docs/introduction-to-image-compression.md`
2. **PNG**: `docs/learning/png/index.md`
3. **JPEG**: `docs/learning/jpg/index.md`
4. **Advanced**: `docs/learning/jpg/trellis.md`

## File Organization

Each file is self-contained with:
- Theory explanation
- Implementation details
- Code examples
- Visual diagrams (ASCII art)
- References to actual source files

## Usage

No build step required - just read the Markdown files in order.

### For Contributors
Read before making changes:
- Understanding the algorithm you're implementing
- Knowing where in the code to make changes
- Understanding trade-offs and design decisions

### For Learning
Follow the structured learning path:
1. Image compression basics
2. PNG encoding details
3. JPEG encoding details
4. Advanced optimization techniques

## Key Features

- **Code Examples**: Real snippets from `src/` packages
- **Algorithm Visualizations**: ASCII diagrams of data flow
- **Performance Notes**: Optimization tips and gotchas
- **Historical Context**: Why certain choices were made

## JIT Index

```bash
# Find JPEG docs
ls docs/learning/jpg/

# Find PNG docs
ls docs/learning/png/

# Search for topics
rg "Trellis" docs/learning/jpg/
rg "DEFLATE" docs/*.md
```

## Common Resources

- **Task Tracking**: `docs/task.md` (project roadmap)
- **Implementation Guide**: `brief.md` (code reading guide)
- **Performance**: `docs/performance-optimization.md`
- **CLI Usage**: `CLI.md` (command reference)

## No Build Required

These are plain text files - edit and view directly in any Markdown viewer.
