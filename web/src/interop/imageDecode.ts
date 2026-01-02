export type DecodeResult = {
  pixels: Uint8Array;
  width: number;
  height: number;
  colorType: number;
  previewUrl: string;
};

export const decodeFile = (file: File): Promise<DecodeResult> =>
  new Promise((resolve, reject) => {
    const reader = new FileReader();

    reader.onload = (e) => {
      try {
        const buffer = e.target?.result;
        if (!(buffer instanceof ArrayBuffer)) {
          reject(new Error("Failed to read file"));
          return;
        }

        const blob = new Blob([buffer], { type: file.type });
        const url = URL.createObjectURL(blob);

        const img = new Image();
        img.onload = () => {
          const canvas = document.createElement("canvas");
          canvas.width = img.width;
          canvas.height = img.height;
          const ctx = canvas.getContext("2d");
          if (!ctx) {
            URL.revokeObjectURL(url);
            reject(new Error("Failed to create canvas context"));
            return;
          }

          ctx.drawImage(img, 0, 0);
          const imageData = ctx.getImageData(0, 0, img.width, img.height);
          const pixels = imageData.data; // Keep as Uint8ClampedArray for WASM

          // PNG color types: 6 = RGBA, 2 = RGB
          const colorType = 6;

          resolve({
            pixels,
            width: img.width,
            height: img.height,
            colorType,
            previewUrl: url,
          });
        };

        img.onerror = () => {
          URL.revokeObjectURL(url);
          reject(new Error("Failed to load image"));
        };

        img.src = url;
      } catch (err) {
        reject(err);
      }
    };

    reader.onerror = () => reject(new Error("Failed to read file"));
    reader.readAsArrayBuffer(file);
  });

