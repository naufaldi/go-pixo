import { zip } from 'fflate';

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

export const downloadAllAsZip = async (items: QueueItem[]): Promise<void> => {
  const done = items.filter(
    (item) => item.status === 'Done' && item.compressedUrl != null
  );

  if (done.length === 0) return;

  // Fetch all compressed blobs in parallel
  const fileEntries = await Promise.all(
    done.map(async (item) => {
      const blob = await fetch(item.compressedUrl as string).then((r) => r.blob());
      const buf = await blob.arrayBuffer();
      const extension = item.compressedExtension ?? '.png';
      return {
        name: buildCompressedFilename(item.file.name, extension),
        data: new Uint8Array(buf),
      };
    })
  );

  // Build fflate file map
  const files: Record<string, Uint8Array> = {};
  for (const entry of fileEntries) {
    files[entry.name] = entry.data;
  }

  return new Promise((resolve, reject) => {
    zip(files, (err, data) => {
      if (err) {
        reject(err);
        return;
      }
      const zipBuffer = data.buffer as ArrayBuffer;
      const url = URL.createObjectURL(
        new Blob([zipBuffer], { type: 'application/zip' })
      );
      try {
        const a = document.createElement('a');
        a.href = url;
        a.download = 'go-pixo-compressed.zip';
        document.body.appendChild(a);
        a.click();
        document.body.removeChild(a);
      } finally {
        URL.revokeObjectURL(url);
        resolve();
      }
    });
  });
};
