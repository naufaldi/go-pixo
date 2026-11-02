# Notice / Attribution

This repository (`go-pixo`) is a work-in-progress Go rewrite inspired by the Rust project `pixo`:

- https://github.com/leerob/pixo

At initialization, some documentation/scaffolding files were copied from the upstream repository for reference.

License: MIT (see `LICENSE`).


## Browser codecs

The web app additionally uses `@jsquash/avif` 2.1.1 (Apache-2.0), `utif` 3.1.0 (MIT), and libheif 1.23.7 with libde265 1.0.16 (LGPL-3.0, via the unmodified `heic-to` 1.6.7 build). Their license texts and source/build references are distributed in `web/public/licenses/`, including instructions for rebuilding with a modified HEIC library. See `web/public/licenses/third-party.txt`.

## Go standard library (lossless JPEG optimizer)

`src/jpeg/coeff_reader.go` and `src/jpeg/coeff_huffman.go` are modified forks of the decoder in Go's `image/jpeg` package, and the JPEG fixtures in `src/jpeg/testdata/` are copied from Go's `src/image/testdata`. They are used under the Go BSD-3-Clause license:

```text
Copyright 2009 The Go Authors.

Redistribution and use in source and binary forms, with or without
modification, are permitted provided that the following conditions are
met:

   * Redistributions of source code must retain the above copyright
notice, this list of conditions and the following disclaimer.
   * Redistributions in binary form must reproduce the above
copyright notice, this list of conditions and the following disclaimer
in the documentation and/or other materials provided with the
distribution.
   * Neither the name of Google LLC nor the names of its
contributors may be used to endorse or promote products derived from
this software without specific prior written permission.

THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS
"AS IS" AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT
LIMITED TO, THE IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR
A PARTICULAR PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT
OWNER OR CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL,
SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT
LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE,
DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY
THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
(INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
```
