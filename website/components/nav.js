import { useState } from 'react'
import Link from 'next/link'
import { Bars2Icon, XMarkIcon } from '@heroicons/react/24/outline'

const links = [
  {
    href: '/docs',
    text: 'Documentation',
  },
  {
    href: '/blog',
    text: 'Blog',
  },
]

export default function Nav() {
  const [open, setOpen] = useState(false)

  return (
    <header
      className={`top-0 z-50 mx-auto flex w-full flex-col items-center bg-white/90 p-4 backdrop-blur-lg transition-colors duration-150 md:sticky ${
        open ? 'fixed bg-white' : 'bg-white/90 backdrop-blur-lg'
      }`}
    >
      <div className='flex w-full max-w-7xl flex-1 items-center justify-between'>
        <Link href='/'>
          <a className='flex-none'>
            <img
              alt='infra logo'
              src='/images/logo.svg'
              className='h-8'
              draggable='false'
            />
          </a>
        </Link>
        <div className='flex'>
          <nav className='relative top-0.5 flex-1 items-baseline text-[15px] font-medium leading-none'>
            {links.map(l => (
              <Link key={l.text} href={l.href}>
                <a className='mx-4 hidden md:inline'>{l.text}</a>
              </Link>
            ))}
            <Link href='/docs/getting-started/quickstart'>
              <a className='group ml-4 inline-flex flex-none items-center rounded-full bg-blue-500 py-2.5 px-3.5 text-[14px] font-semibold leading-3 text-white shadow-sm transition-colors hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2'>
                Get Started{' '}
                <span className='ml-1 transition-transform group-hover:translate-x-0.5'>
                  ›
                </span>
              </a>
            </Link>
          </nav>
          <div
            className='-mr-4 flex flex-row items-center space-x-12 py-2 px-4 text-base text-black md:hidden'
            onClick={() => setOpen(!open)}
          >
            {open ? (
              <XMarkIcon className='h-6 w-6' />
            ) : (
              <Bars2Icon className='h-6 w-6' />
            )}
          </div>
        </div>
      </div>
      {open && (
        <nav className='absolute top-full flex h-screen w-full flex-1 flex-col space-y-10 bg-white p-4 text-[15px] font-medium leading-none backdrop-blur-lg md:hidden'>
          {links.map(l => (
            <Link key={l.text} href={l.href}>
              <a onClick={() => setOpen(false)}>{l.text}</a>
            </Link>
          ))}
        </nav>
      )}
    </header>
  )
}
