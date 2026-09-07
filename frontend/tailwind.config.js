/** @type {import('tailwindcss').Config} */
export default {
  content: ['./index.html', './src/**/*.{js,ts,jsx,tsx}'],
  darkMode: 'class',
  theme: {
    extend: {
      fontFamily: {
        sans: ['Inter', 'system-ui', 'sans-serif'],
        display: ['Sora', 'Inter', 'system-ui', 'sans-serif'],
      },
      colors: {
        navy: {
          950: 'rgb(var(--navy-950) / <alpha-value>)',
          900: 'rgb(var(--navy-900) / <alpha-value>)',
          850: 'rgb(var(--navy-850) / <alpha-value>)',
          800: 'rgb(var(--navy-800) / <alpha-value>)',
          700: 'rgb(var(--navy-700) / <alpha-value>)',
          600: 'rgb(var(--navy-600) / <alpha-value>)',
        },
        brand: {
          DEFAULT: 'rgb(var(--brand) / <alpha-value>)',
          light: 'rgb(var(--brand-light) / <alpha-value>)',
          dark: 'rgb(var(--brand-dark) / <alpha-value>)',
          glow: 'rgb(var(--brand-glow) / <alpha-value>)',
        },
        gold: {
          DEFAULT: 'rgb(var(--gold) / <alpha-value>)',
          light: 'rgb(var(--gold-light) / <alpha-value>)',
          dark: 'rgb(var(--gold-dark) / <alpha-value>)',
        },
        pitch: { 900: '#0b0f19', 800: '#141b2d', 700: '#1a2338' },
        accent: { DEFAULT: '#10b981', dark: '#059669' },
      },
      boxShadow: {
        glow: '0 0 40px -10px rgba(16, 185, 129, 0.35)',
        card: '0 8px 32px -8px rgba(0, 0, 0, 0.12)',
        'card-dark': '0 8px 32px -8px rgba(0, 0, 0, 0.5)',
        logo: '0 4px 24px rgba(16, 185, 129, 0.35)',
      },
    },
  },
  plugins: [],
};
