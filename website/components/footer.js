import Link from 'next/link'

const footerLinks = [
  {
    href: '/docs/reference/security',
    text: 'Security',
  },
  {
    href: '/about',
    text: 'About',
  },
  {
    href: 'https://www.ycombinator.com/companies/infra/jobs',
    text: 'Work with us',
  },
  {
    href: 'https://github.com/infrahq/infra',
    text: 'GitHub',
  },
  {
    href: 'https://twitter.com/infrahq',
    text: 'Twitter',
  },
]

export default function Footer() {
  return (
    <footer className='relative z-30 mx-auto flex w-full max-w-screen-2xl flex-none flex-col justify-between bg-black px-6 py-8 md:flex-row md:px-8'>
      <nav className='my-8 flex flex-1 flex-col items-baseline space-x-0 space-y-8 md:my-0 md:flex-row md:space-y-0 md:space-x-8 md:text-sm'>
        <Link href='/'>
          <a>
            <img
              alt='infra logo'
              src='/images/logo-white.svg'
              className='-mb-px h-5 md:h-4'
              draggable='false'
            />
          </a>
        </Link>
        {footerLinks.map(l => (
          <Link key={l.text} href={l.href}>
            <a className='text-sm text-gray-300'>{l.text}</a>
          </Link>
        ))}
      </nav>
      <span className='text-sm text-gray-400'>
        © {new Date().getUTCFullYear()} Infra Technologies, Inc.
      </span>
    </footer>
  )
}
