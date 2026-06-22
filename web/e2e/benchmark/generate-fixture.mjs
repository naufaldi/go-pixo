import { chromium } from '@playwright/test';
import { mkdir, writeFile } from 'node:fs/promises';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';

const benchmarkDir = dirname(fileURLToPath(import.meta.url));
const fixturePath = join(
  benchmarkDir,
  '..',
  'fixtures',
  'benchmark',
  'generated-photo-like-1024x768.png',
);

async function generatePng() {
  const browser = await chromium.launch();
  const page = await browser.newPage();

  try {
    return await page.evaluate(async () => {
      const width = 1024;
      const height = 768;
      const canvas = document.createElement('canvas');
      canvas.width = width;
      canvas.height = height;
      const ctx = canvas.getContext('2d');
      if (ctx == null) throw new Error('failed to create canvas context');

      const sky = ctx.createLinearGradient(0, 0, width, height);
      sky.addColorStop(0, '#0f5e9c');
      sky.addColorStop(0.35, '#37a7c8');
      sky.addColorStop(0.68, '#f1c27d');
      sky.addColorStop(1, '#243447');
      ctx.fillStyle = sky;
      ctx.fillRect(0, 0, width, height);

      let seed = 392017;
      const random = () => {
        seed = (seed * 1664525 + 1013904223) >>> 0;
        return seed / 0xffffffff;
      };

      const imageData = ctx.getImageData(0, 0, width, height);
      for (let y = 0; y < height; y += 1) {
        for (let x = 0; x < width; x += 1) {
          const i = (y * width + x) * 4;
          const wave = Math.sin(x / 24) * 18 + Math.cos(y / 31) * 16;
          const noise = Math.floor((random() - 0.5) * 42);
          imageData.data[i] = Math.max(0, Math.min(255, imageData.data[i] + wave + noise));
          imageData.data[i + 1] = Math.max(0, Math.min(255, imageData.data[i + 1] + noise));
          imageData.data[i + 2] = Math.max(0, Math.min(255, imageData.data[i + 2] - wave + noise));
          imageData.data[i + 3] = 255;
        }
      }
      ctx.putImageData(imageData, 0, 0);

      ctx.fillStyle = 'rgba(12, 16, 24, 0.72)';
      ctx.fillRect(72, 86, 340, 194);
      ctx.fillStyle = 'rgba(255, 255, 255, 0.88)';
      for (let i = 0; i < 8; i += 1) {
        ctx.fillRect(104, 126 + i * 18, 230 + (i % 3) * 28, 8);
      }

      ctx.fillStyle = '#fb7185';
      ctx.fillRect(640, 108, 220, 220);
      ctx.fillStyle = '#22c55e';
      ctx.fillRect(686, 154, 220, 220);
      ctx.fillStyle = '#facc15';
      ctx.fillRect(732, 200, 220, 220);

      ctx.strokeStyle = 'rgba(255, 255, 255, 0.9)';
      ctx.lineWidth = 4;
      for (let x = 0; x < width; x += 64) {
        ctx.beginPath();
        ctx.moveTo(x, 0);
        ctx.lineTo(width - x / 3, height);
        ctx.stroke();
      }

      ctx.fillStyle = 'rgba(15, 23, 42, 0.8)';
      ctx.fillRect(120, 544, 784, 104);
      ctx.fillStyle = '#ffffff';
      ctx.font = '700 42px system-ui, sans-serif';
      ctx.fillText('GO-PIXO BENCHMARK', 164, 604);
      ctx.font = '500 24px system-ui, sans-serif';
      ctx.fillText('gradient + noise + edges + text blocks', 166, 636);

      const dataUrl = canvas.toDataURL('image/png');
      return dataUrl.slice(dataUrl.indexOf(',') + 1);
    });
  } finally {
    await browser.close();
  }
}

await mkdir(dirname(fixturePath), { recursive: true });
const base64 = await generatePng();
const bytes = Buffer.from(base64, 'base64');
await writeFile(fixturePath, bytes);

console.log(`Generated ${fixturePath}`);
console.log(`Size ${bytes.length} bytes`);
