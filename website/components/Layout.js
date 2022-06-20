import Nav from './Nav'
import Footer from './Footer'

export default function ({ children }) {
  return (
    <div className='flex flex-col overflow-x-hidden min-h-full'>
      <Nav />
      {children}
      <Footer />
    </div>
  )
}
