module.exports = {
  content: [
    './pages/**/*.{js,ts,jsx,tsx}',
    './components/**/*.{js,ts,jsx,tsx}'
  ],
  theme: {
    extend: {
      colors: {
        black: '#0A0E12',
        gray: {
          50: '#FDFDFE',
          100: '#F3F4F6',
          200: '#E5E7EB',
          300: '#D2D5DA',
          400: '#9DA3AE',
          500: '#747B8B',
          600: '#4D5562',
          700: '#394150',
          800: '#222833',
          900: '#151A1E'
        }
      },
      fontSize: {
        xs: ['13px', '16px'],
        '2xs': ['12px', '15px'],
        '3xs': ['11px', '13px']
      },
      animation: {
        'spin-fast': 'spin 0.75s linear infinite'
      },
      fontFamily: {
        sans: ['SF Pro Text', 'ui-sans-serif', 'sans-serif'],
        mono: ['SF Mono', 'ui-monospace', 'monospace']
      }
    }
  }
}
