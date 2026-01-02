type QueueItem = {
  file: File;
  status: { tag: string };
  compressedUrl: string | null;
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
    if (item.status?.tag === "Done" && item.compressedUrl != null) {
      setTimeout(() => {
        void downloadBlob(item.compressedUrl as string, `compressed_${item.file.name}`);
      }, index * 100);
    }
  });
};

