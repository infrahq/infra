import Nav from './nav'
import Footer from './footer'

export default function Layout({ children }) {
  return (
    <div className='flex min-h-full flex-col'>
      <Nav />
      <div className='flex flex-1 flex-col'>{children}</div>
      <Footer />
    </div>
  )
}
