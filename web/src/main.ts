import './style.css';
import { initWasm, encodePng } from './wasm';

const fileInput = document.getElementById('file-input') as HTMLInputElement;
const dropZone = document.getElementById('drop-zone') as HTMLDivElement;
const status = document.getElementById('status') as HTMLDivElement;
const previewContainer = document.getElementById('preview-container') as HTMLDivElement;
const originalPreview = document.getElementById('original-preview') as HTMLImageElement;
const compressedPreview = document.getElementById('compressed-preview') as HTMLImageElement;
const originalInfo = document.getElementById('original-info') as HTMLParagraphElement;
const compressedInfo = document.getElementById('compressed-info') as HTMLParagraphElement;
const downloadBtn = document.getElementById('download-btn') as HTMLButtonElement;
const actions = document.getElementById('actions') as HTMLDivElement;

let compressedBlob: Blob | null = null;
let compressedFileName = 'compressed.png';

// Initialize WASM
status.classList.remove('hidden');
status.textContent = 'Initializing WASM...';

initWasm()
  .then(() => {
    status.classList.add('hidden');
    console.log('WASM ready');
  })
  .catch(err => {
    status.textContent = 'Failed to load WASM';
    status.classList.add('bg-red-50', 'text-red-700');
    console.error(err);
  });

// Event Listeners
dropZone.addEventListener('click', () => fileInput.click());

dropZone.addEventListener('dragover', (e) => {
  e.preventDefault();
  dropZone.classList.add('border-blue-400', 'bg-blue-50');
});

dropZone.addEventListener('dragleave', () => {
  dropZone.classList.remove('border-blue-400', 'bg-blue-50');
});

dropZone.addEventListener('drop', (e) => {
  e.preventDefault();
  dropZone.classList.remove('border-blue-400', 'bg-blue-50');
  const files = e.dataTransfer?.files;
  if (files && files.length > 0) {
    handleFile(files[0]);
  }
});

fileInput.addEventListener('change', () => {
  if (fileInput.files && fileInput.files.length > 0) {
    handleFile(fileInput.files[0]);
  }
});

const handleFile = async (file: File) => {
  if (!file.type.startsWith('image/')) return;

  status.classList.remove('hidden');
  status.textContent = 'Processing...';
  previewContainer.classList.add('hidden');
  actions.classList.add('hidden');

  const reader = new FileReader();
  reader.onload = async (e) => {
    const dataUrl = e.target?.result as string;
    originalPreview.src = dataUrl;
    originalInfo.textContent = `${(file.size / 1024).toFixed(2)} KB`;

    const img = new Image();
    img.onload = async () => {
      const canvas = document.createElement('canvas');
      canvas.width = img.width;
      canvas.height = img.height;
      const ctx = canvas.getContext('2d');
      if (!ctx) return;

      ctx.drawImage(img, 0, 0);
      const imageData = ctx.getImageData(0, 0, img.width, img.height);
      const pixels = new Uint8Array(imageData.data.buffer);

      try {
        const startTime = performance.now();
        // Call WASM
        const result = encodePng(pixels, img.width, img.height, 6, 1, false);
        const endTime = performance.now();
        console.log(`Compression took ${(endTime - startTime).toFixed(2)}ms`);

        compressedBlob = new Blob([result as any], { type: 'image/png' });
        compressedFileName = `compressed_${file.name.split('.')[0]}.png`;
        const compressedUrl = URL.createObjectURL(compressedBlob);
        
        compressedPreview.src = compressedUrl;
        compressedInfo.textContent = `${(compressedBlob.size / 1024).toFixed(2)} KB (${((1 - compressedBlob.size / file.size) * 100).toFixed(1)}% reduction)`;

        status.classList.add('hidden');
        previewContainer.classList.remove('hidden');
        actions.classList.remove('hidden');
      } catch (err) {
        status.textContent = 'Error during compression';
        status.classList.add('bg-red-50', 'text-red-700');
        console.error(err);
      }
    };
    img.src = dataUrl;
  };
  reader.readAsDataURL(file);
};

downloadBtn.addEventListener('click', () => {
  if (compressedBlob) {
    const url = URL.createObjectURL(compressedBlob);
    const a = document.createElement('a');
    a.href = url;
    a.download = compressedFileName;
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
    URL.revokeObjectURL(url);
  }
});
