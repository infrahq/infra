module.exports = {
  content: [
    './pages/**/*.{js,ts,jsx,tsx}',
    './components/**/*.{js,ts,jsx,tsx}',
  ],
  theme: {
    extend: {
      colors: {
        black: '#0A0E12',
      },
      fontFamily: {
        sans: ['neuzeit-grotesk', 'sans-serif'],
        mono: ['input-mono', 'monospace'],
      },
    },
  },
  plugins: [require('@tailwindcss/typography')],
}
