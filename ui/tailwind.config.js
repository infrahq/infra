function withOpacityValue(variable) {
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
          'light': withOpacityValue('228 64 255'),
          dark: '#CB2EEC'
        },
        gray: {
          light: '#B2B2B2'
        }
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
