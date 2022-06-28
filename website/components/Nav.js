import { useState } from 'react'
import Link from 'next/link'
import { MenuIcon, XIcon } from '@heroicons/react/outline'

const links = [
  {
    href: '/docs',
    text: 'Documentation',
  },
  {
    href: 'https://blog.infrahq.com',
    text: 'Blog',
  },
  {
    href: '/about',
    text: 'About',
  },
]

export default function Nav({ docs = false }) {
  const [open, setOpen] = useState(false)

  return (
    <header className='sticky top-0 z-50 mx-auto flex w-full max-w-screen-2xl flex-col items-center bg-black/90 px-6 py-5 text-white backdrop-blur-lg md:px-8'>
      <div className='flex w-full flex-1 items-center justify-between'>
        <Link href='/'>
          <a className='flex'>
            <img
              alt='infra logo'
              src='/images/logo-white.svg'
              className='mb-1 h-5'
              draggable='false'
            />
            {docs && (
              <img
                alt='infra docs'
                className='ml-0.5 h-5'
                src='/images/logo-docs.svg'
              />
            )}
          </a>
        </Link>
        <nav className='hidden flex-row items-center space-x-12 md:flex'>
          {links.map(l => (
            <Link key={l.text} href={l.href}>
              <a>{l.text}</a>
            </Link>
          ))}
          <Link href='https://github.com/infrahq/infra'>
            <a>
              <div className='overflow-hidden rounded-full bg-gradient-to-tr from-cyan-100 to-pink-300'>
                <button className='m-px flex items-center rounded-full bg-black py-1.5 pl-2 pr-3 hover:bg-gray-900'>
                  <img
                    alt='github logo'
                    className='h-5 pr-2'
                    src='/images/github.svg'
                  />{' '}
                  Open in GitHub
                </button>
              </div>
            </a>
          </Link>
        </nav>
        <div
          className='-mr-4 flex flex-row items-center space-x-12 py-2 px-4 text-base md:hidden'
          onClick={() => setOpen(!open)}
        >
          {open ? (
            <XIcon className='h-6 w-6 text-zinc-200' />
          ) : (
            <MenuIcon className='h-6 w-6 text-zinc-200' />
          )}
        </div>
      </div>
      <nav
        className={`relative z-10 flex w-full flex-col justify-around space-y-6 py-5 text-lg font-thin ${
          open ? '' : 'hidden'
        }`}
      >
        {links.map(l => (
          <Link key={l.text} href={l.href}>
            <a onClick={() => setOpen(false)} className='w-full py-2'>
              {l.text}
            </a>
          </Link>
        ))}
        <Link href='https://github.com/infrahq/infra'>
          <a>
            <div className='inline-flex overflow-hidden rounded-full bg-gradient-to-tr from-cyan-100 to-pink-300'>
              <button className='m-px flex items-center rounded-full bg-black py-1.5 pl-3 pr-4 hover:bg-gray-900'>
                <img
                  alt='github logo'
                  className='h-5 pr-2'
                  src='/images/github.svg'
                />{' '}
                Open in GitHub
              </button>
            </div>
          </a>
        </Link>
      </nav>
    </header>
  )
}
