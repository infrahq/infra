import '../styles/globals.css'
import type { AppProps } from 'next/app'
import { AuthContextProvider } from '../store/AuthContext'

function App({ Component, pageProps }: AppProps) {
  return (
    <AuthContextProvider>
      <Component {...pageProps} />
    </AuthContextProvider>
  )
}

export default App
