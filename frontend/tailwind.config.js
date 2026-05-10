/** @type {import('tailwindcss').Config} */
export default {
  content: [
    "./index.html",
    "./src/**/*.{js,ts,jsx,tsx}",
  ],
  theme: {
    extend: {
      colors: {
        // EVE Frontier palette
        background: '#000000',
        surface: '#0b0b0b',
        panel: '#0f0f0f',
        elevated: '#161613',
        text: {
          main: '#fafae5',
          muted: '#9e9e91',
          dim: '#6b6b63',
        },
        brand: {
          DEFAULT: '#adff00',
          dim: '#7fbf00',
          glow: '#adff0033',
        },
        accent: {
          orange: '#ff4700',
          orangeDim: '#ff550075',
          orangeFaint: '#ff550030',
        },
        signal: {
          green: '#adff00',
          amber: '#ff4700',
          red: '#c20046',
          blue: '#0c27db',
        },
        line: {
          DEFAULT: 'hsla(60, 68%, 94%, 0.145)',
          strong: 'hsla(60, 68%, 94%, 0.3)',
        },
      },
      fontFamily: {
        mono: ['"JetBrains Mono"', 'Menlo', 'Monaco', 'Courier New', 'monospace'],
        display: ['"Inter"', '"Helvetica Neue"', 'Arial', 'sans-serif'],
      },
      boxShadow: {
        'brand-glow': '0 0 0 1px #adff00, 0 0 12px 0 #adff0055',
        'orange-glow': '0 0 0 1px #ff4700, 0 0 12px 0 #ff470055',
      },
    },
  },
  plugins: [],
}
