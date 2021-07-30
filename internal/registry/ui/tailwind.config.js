module.exports = {
  purge: [
    './pages/**/*.{js,ts,jsx,tsx}',
    './components/**/*.{js,ts,jsx,tsx}',
    './layouts/**/*.{js,ts,jsx,tsx}'
  ],
  darkMode: false,
  theme: {
    extend: {
      colors: {
        blue: {
          50: '#EFF7FF',
          100: '#DCEDFD',
          200: '#D0E6FF',
          300: '#A3CEF5',
          400: '#66A8F7',
          500: '#3B85F5',
          600: '#0069FF',
          700: '#0369E1',
          800: '#0245AA',
          900: '#013E85'
        },
      },
      transitionProperty: {
        'height': 'height'
      },
    }
  },
  variants: {
    extend: {},
  },
  plugins: [],
}
