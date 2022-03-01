import '../styles/globals.css'
import { AuthContextProvider } from '../store/AuthContext'

function App ({ Component, pageProps }) {
  return (
    <AuthContextProvider>
      <Component {...pageProps} />
    </AuthContextProvider>
  )
}

export default App
