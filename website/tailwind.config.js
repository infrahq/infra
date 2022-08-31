module.exports = {
  content: [
    './pages/**/*.{js,ts,jsx,tsx}',
    './components/**/*.{js,ts,jsx,tsx}',
  ],
  theme: {
    extend: {
      colors: {
        black: '#0A0E12',
        blue: {
          50: '#F1F7FE',
          100: '#DEE9FC',
          200: '#C4DAFB',
          300: '#95C0FD',
          400: '#5F9DFB',
          500: '#1D67F9',
          600: '#0F60FF',
          700: '#1159E9',
          800: '#0C44B5',
          900: '#0B3C9E',
        },
      },
      fontFamily: {
        sans: [
          '-apple-system',
          'BlinkMacSystemFont',
          'segoe ui',
          'helvetica neue',
          'helvetica',
          'Ubuntu',
          'roboto',
          'arial',
          'sans-serif',
        ],
        mono: ['SF Mono', 'Menlo', 'Consolas', 'monospace'],
      },
    },
  },
  plugins: [require('@tailwindcss/typography')],
}
