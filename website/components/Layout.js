import Nav from './Nav'
import Footer from './Footer'

export default function Layout({ children }) {
  return (
    <div className='flex min-h-full flex-col overflow-x-hidden'>
      <Nav />
      {children}
      <Footer />
    </div>
  )
}
