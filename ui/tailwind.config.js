function withOpacityValue (variable) {
  return ({ opacityValue }) => {
    if (opacityValue === undefined) {
      return `rgb(${variable})`
    }
    return `rgb(${variable} / ${opacityValue})`
  }
}

module.exports = {
  content: [
    './pages/**/*.{js,ts,jsx,tsx}',
    './components/**/*.{js,ts,jsx,tsx}'
  ],
  theme: {
    extend: {
      colors: {
        black: '#0A0E12',
        pink: {
          100: '#DECAFF',
          200: '#F4E2FF',
          300: '#EB91C7',
          light: withOpacityValue('228 64 255'),
          dark: '#CB2EEC'
        },
        gray: {
          200: '#78828A',
          300: '#B2B2B2',
          350: '#2A2D34',
          400: '#868C9A',
          500: '#6B6674',
          800: '#32393F',
          900: '#1C2027',
          950: '#292E33'
        },
        purple: {
          50: '#F4E2FF'
        }
      },
      fontSize: {
        label: ['11px', '0px'],
        title: ['13px', '16px'],
        header: ['16px', '19px'],
        note: ['10px', '12px'],
        name: ['12px', '15px'],
        subtitle: ['12px', '0px'],
        paragraph: ['12px', '22px'],
        secondary: ['10px', '12px']
      },
      transitionProperty: {
        size: 'height, padding, background'
      },
      animation: {
        'spin-fast': 'spin 0.75s linear infinite'
      },
      fontFamily: {
        sans: ['SF Pro Text', 'BlinkMacSystemFont', 'Segoe UI', 'Ubuntu', 'sans-serif'],
        mono: ['SF Mono', 'monospace']
      }
    }
  }
}
