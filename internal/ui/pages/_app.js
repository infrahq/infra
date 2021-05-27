import '../styles/global.css'

function App({ Component, pageProps }) {
  return <Component {...pageProps} className='antialiased' />
}

export default App
