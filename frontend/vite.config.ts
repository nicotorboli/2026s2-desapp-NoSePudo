import react from '@vitejs/plugin-react';
import { defineConfig } from 'vitest/config';

export default defineConfig({
  plugins: [react()],
  test: {
    // Los tests de componentes necesitan DOM; los de api/ no, pero jsdom no molesta.
    environment: 'jsdom',
    coverage: {
      provider: 'v8',
      // lcov es el formato que consume Sonar (ver sonar-project.properties).
      reporter: ['text', 'lcov'],
      include: ['src/**/*.{ts,tsx}'],
      // main.tsx es el entrypoint: una llamada a createRoot().render().
      // Testearlo seria testear React, no codigo propio.
      exclude: ['src/main.tsx'],
    },
  },
});
