/** @type {import('tailwindcss').Config} */
export default {
  content: ['./index.html', './src/**/*.{js,ts,jsx,tsx}'],
  theme: {
    extend: {
      colors: {
        pitch: { 900: '#0a1f0a', 800: '#122812', 700: '#1a3a1a' },
        accent: { DEFAULT: '#22c55e', dark: '#16a34a' },
      },
    },
  },
  plugins: [],
}
