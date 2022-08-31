import Link from 'next/link'

const footerLinks = [
  {
    href: '/docs',
    text: 'Documentation',
  },
  {
    href: '/blog',
    text: 'Blog',
  },
  {
    href: '/docs/reference/security',
    text: 'Security',
  },
  {
    href: 'https://www.ycombinator.com/companies/infra/jobs',
    text: 'Work with us',
  },
  {
    href: 'https://status.infrahq.com',
    text: 'Status',
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
    <footer className='w-full bg-gray-50'>
      <div className='relative z-30 mx-auto flex w-full max-w-7xl flex-none flex-col justify-between bg-gray-50 p-4 text-sm md:flex-row md:text-xs'>
        <nav className='mt-2 mb-8 flex flex-1 flex-col items-baseline space-x-0 space-y-8 font-medium tracking-tight text-gray-500 md:my-0 md:mb-0 md:flex-row md:space-y-0 md:space-x-6'>
          {footerLinks.map(l => (
            <Link key={l.text} href={l.href}>
              <a
                className='hover:text-gray-700'
                rel='noopener noreferrer'
                target={l.href?.startsWith('/') ? '' : '_blank'}
              >
                {l.text}
              </a>
            </Link>
          ))}
        </nav>
        <span className='text-xs font-semibold tracking-tight text-gray-400 md:text-[11px]'>
          © {new Date().getUTCFullYear()} Infra Technologies, Inc.
        </span>
      </div>
    </footer>
  )
}
