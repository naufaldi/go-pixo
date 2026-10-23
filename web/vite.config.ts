import { defineConfig } from 'vite';
import tailwindcss from '@tailwindcss/vite';
import react from '@vitejs/plugin-react';
import { offlinePlugin } from './build/offline';

export default defineConfig({
  // Codec imports stay lazy inside the compression worker.
  worker: { format: 'es' },
  plugins: [
    react(),
    tailwindcss(),
    offlinePlugin(),
  ],
});
