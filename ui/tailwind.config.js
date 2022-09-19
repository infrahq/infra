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
      fontSize: {
        xs: ['13px', '16px'],
        '2xs': ['12px', '15px'],
        '3xs': ['11px', '13px'],
        '4xs': ['10px', '12px'],
      },
      animation: {
        'spin-fast': 'spin 0.75s linear infinite',
      },
      fontFamily: {
        sans: ['sans', 'sans-serif'],
        display: ['display', 'sans-serif'],
        mono: ['mono', 'monospace'],
      },
    },
  },
  plugins: [require('@tailwindcss/forms')],
}
