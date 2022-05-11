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
          300: '#EB91C7',
          light: withOpacityValue('228 64 255'),
          dark: '#CB2EEC'
        },
        gray: {
          200: '#78828A',
          300: '#B2B2B2',
          950: '#292E33'
        },
        purple: {
          50: '#F4E2FF'
        }
      },
      fontSize: {
        label: ['11px', '0px'],
        name: ['12px', '15px'],
        title: ['13px', '16px'],
        header: ['16px', '19px']
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
