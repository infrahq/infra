import Nav from './Nav'
import Footer from './Footer'

export default function ({ children }) {
  return (
    <div className='h-screen flex flex-col overflow-x-hidden'>
      <Nav />
      {children}
      <Footer />
    </div>
  )
}
