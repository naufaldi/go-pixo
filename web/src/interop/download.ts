import { Zip, ZipPassThrough } from 'fflate';

// ReScript simple variants (Done, Pending, etc.) compile to bare strings.
// Payload variants (Error(msg)) compile to { TAG: string; _0: string }.
type QueueItemStatus = string | { TAG: string; _0: string };

type QueueItem = {
  file: File;
  status: QueueItemStatus;
  compressedUrl: string | null;
  compressedExtension?: string | null;
};

export const buildCompressedFilename = (
  originalName: string,
  extension: string,
): string => {
  const dot = originalName.lastIndexOf('.');
  const base = dot > 0 ? originalName.slice(0, dot) : originalName;
  return `compressed_${base}${extension}`;
};

export const downloadBlob = async (blobUrl: string, filename: string): Promise<void> => {
  const blob = await fetch(blobUrl).then((r) => r.blob());
  const url = URL.createObjectURL(blob);
  try {
    const a = document.createElement("a");
    a.href = url;
    a.download = filename;
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
  } finally {
    URL.revokeObjectURL(url);
  }
};

export const downloadAll = (items: QueueItem[]): void => {
  items.forEach((item, index) => {
    if (item.status === "Done" && item.compressedUrl != null) {
      const extension = item.compressedExtension ?? '.png';
      setTimeout(() => {
        void downloadBlob(
          item.compressedUrl as string,
          buildCompressedFilename(item.file.name, extension),
        );
      }, index * 100);
    }
  });
};

// Bound both the input allocation and each synchronous CRC pass. Blob URL
// responses stay as Blobs; only this slice becomes JS bytes at a time.
const ZIP_CHUNK_BYTES = 64 * 1024;
// fflate writes classic ZIP fields, without ZIP64 support. Reject overflow
// instead of emitting wrapped entry counts, file sizes, or archive offsets.
const ZIP_MAX_BYTES = 0xffff_ffff;

const withAbort = <T>(operation: Promise<T>, signal?: AbortSignal): Promise<T> => {
  if (!signal) return operation;
  return new Promise((resolve, reject) => {
    const onAbort = () => {
      signal.removeEventListener('abort', onAbort);
      reject(signal.reason);
    };
    signal.addEventListener('abort', onAbort, { once: true });
    // Also observe late rejections from operations which ignore cancellation.
    operation.then(value => {
      signal.removeEventListener('abort', onAbort);
      if (signal.aborted) reject(signal.reason);
      else resolve(value);
    }, error => {
      signal.removeEventListener('abort', onAbort);
      reject(error);
    });
    if (signal.aborted) onAbort();
  });
};

export const downloadAllAsZip = async (
  items: QueueItem[],
  signal?: AbortSignal,
): Promise<void> => {
  signal?.throwIfAborted();
  const done = items.filter(
    (item) => item.status === 'Done' && item.compressedUrl != null
  );
  if (done.length === 0) return;
  if (done.length > 65_535) throw new Error('Batch exceeds the ZIP entry limit (65,535 files)');

  const chunks: BlobPart[] = [];
  let complete = false;
  let archiveBytes = 0;
  const archive = new Zip((error, chunk, final) => {
    if (error) throw error;
    archiveBytes += chunk.byteLength;
    if (archiveBytes > ZIP_MAX_BYTES) throw new Error('Archive exceeds the ZIP size limit (4 GiB)');
    // fflate and Blob streams produce ArrayBuffer-backed chunks. Keep each
    // chunk as a Blob part instead of flattening a second archive-sized array.
    chunks.push(chunk as Uint8Array<ArrayBuffer>);
    complete = final;
  });
  let reader: ReadableStreamDefaultReader<Uint8Array> | undefined;
  const releaseReader = (cancel: boolean) => {
    const current = reader;
    reader = undefined;
    if (!current) return;
    try {
      if (cancel) void current.cancel().catch(() => {});
    } finally {
      current.releaseLock();
    }
  };

  try {
    const names = new Set<string>();
    for (const item of done) {
      signal?.throwIfAborted();
      const response = await withAbort(fetch(item.compressedUrl as string, { signal }), signal);
      signal?.throwIfAborted();
      const blob = await withAbort(response.blob(), signal);
      signal?.throwIfAborted();
      if (blob.size > ZIP_MAX_BYTES) throw new Error('File exceeds the ZIP size limit (4 GiB)');

      const filename = buildCompressedFilename(item.file.name, item.compressedExtension ?? '.png');
      const dot = filename.lastIndexOf('.');
      const base = dot > 0 ? filename.slice(0, dot) : filename;
      const extension = dot > 0 ? filename.slice(dot) : '';
      let name = filename;
      let suffix = 2;
      while (names.has(name)) name = `${base} (${suffix++})${extension}`;
      names.add(name);

      // Stored entries preserve the already-compressed image bytes, without
      // redundant DEFLATE work or a worker per file. Some archives will be larger.
      const entry = new ZipPassThrough(name);
      archive.add(entry);
      for (let offset = 0; offset < blob.size; offset += ZIP_CHUNK_BYTES) {
        signal?.throwIfAborted();
        reader = blob.slice(offset, Math.min(offset + ZIP_CHUNK_BYTES, blob.size)).stream().getReader();
        while (true) {
          const { done, value } = await withAbort(reader.read(), signal);
          signal?.throwIfAborted();
          if (done) break;
          entry.push(value);
        }
        releaseReader(false);
      }
      entry.push(new Uint8Array(), true);
    }
    archive.end();
    signal?.throwIfAborted();
    if (!complete) throw new Error('ZIP archive did not finish');

    const url = URL.createObjectURL(new Blob(chunks, { type: 'application/zip' }));
    const anchor = document.createElement('a');
    try {
      signal?.throwIfAborted();
      anchor.href = url;
      anchor.download = 'go-pixo-compressed.zip';
      document.body.appendChild(anchor);
      anchor.click();
    } finally {
      anchor.remove();
      URL.revokeObjectURL(url);
    }
  } finally {
    releaseReader(true);
    archive.terminate();
    chunks.length = 0;
  }
};
