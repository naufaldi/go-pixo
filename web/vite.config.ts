import { defineConfig } from 'vite';
import tailwindcss from '@tailwindcss/vite';
import react from '@vitejs/plugin-react';
import { offlinePlugin } from './build/offline';

export default defineConfig({
  // Codec imports stay lazy inside the compression worker.
  worker: { format: 'es' },
  // These lazy worker imports escape the initial HTML dependency scan. Discover
  // them before decoding so dev optimization cannot invalidate in-flight URLs.
  optimizeDeps: { include: ['pako', 'utif', '@jsquash/avif/codec/enc/avif_enc.js',
    '@jsquash/avif/codec/dec/avif_dec.js', '@jsquash/avif/meta.js'] },
  plugins: [
    react(),
    tailwindcss(),
    offlinePlugin(),
  ],
});
