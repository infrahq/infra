import Link from 'next/link'

const footerLinks = [{
  href: '/docs/reference/security',
  text: 'Security'
}, {
  href: '/about',
  text: 'About'
}, {
  href: 'https://jobs.lever.co/infra-hq',
  text: 'Work with us'
}, {
  href: 'https://github.com/infrahq/infra',
  text: 'GitHub'
}, {
  href: 'https://twitter.com/infrahq',
  text: 'Twitter'
}]

export default function () {
  return (
    <footer className='flex-none flex flex-col md:flex-row px-6 md:px-8 w-full justify-between py-8 relative max-w-screen-2xl mx-auto z-30 bg-black'>
      <nav className='flex flex-col items-baseline space-x-0 space-y-8 my-8 md:my-0 md:space-y-0 md:space-x-8 md:flex-row flex-1 md:text-sm'>
        <Link href='/'>
          <a>
            <img src='/images/logo-white.svg' className='-mb-px h-5 md:h-4' draggable='false' />
          </a>
        </Link>
        {footerLinks.map(l => (
          <Link key={l.text} href={l.href}>
            <a className='text-gray-300 text-sm'>{l.text}</a>
          </Link>
        ))}
      </nav>
      <span className='text-gray-400 text-sm'>© {new Date().getUTCFullYear()} Infra Technologies, Inc.</span>
    </footer>
  )
}
