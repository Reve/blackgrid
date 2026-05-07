/** @type {import('tailwindcss').Config} */
export default {
  content: [
    "./index.html",
    "./src/**/*.{js,ts,jsx,tsx}",
  ],
  theme: {
    extend: {
      colors: {
        background: '#0a0a0a',
        surface: '#111111',
        panel: '#1a1a1a',
        text: {
          main: '#e5e5e5',
          muted: '#a3a3a3',
        },
        signal: {
          green: '#10b981',
          amber: '#f59e0b',
          red: '#ef4444',
        }
      },
      fontFamily: {
        mono: ['"JetBrains Mono"', 'Menlo', 'Monaco', 'Courier New', 'monospace'],
      },
    },
  },
  plugins: [],
}
