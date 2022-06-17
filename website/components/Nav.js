import { useState } from 'react'
import Link from 'next/link'
import { MenuIcon, XIcon } from '@heroicons/react/outline'

const links = [{
  href: '/docs',
  text: 'Documentation'
}, {
  href: 'https://blog.infrahq.com',
  text: 'Blog'
}, {
  href: '/about',
  text: 'About'
}]

export default function ({ docs = false }) {
  const [open, setOpen] = useState(false)

  return (
    <header className='sticky top-0 flex flex-col px-6 md:px-8 py-5 items-center text-white bg-black/90 backdrop-blur-lg z-50 max-w-screen-2xl w-full mx-auto'>
      <div className='flex flex-1 items-center justify-between w-full'>
        <Link href='/'>
          <a className='flex'>
            <img src='/images/logo-white.svg' className='h-5 mb-1' draggable='false' />{docs && <img className='h-5 ml-0.5' src='/images/logo-docs.svg' />}
          </a>
        </Link>
        <nav className='hidden md:flex flex-row items-center space-x-12'>
          {links.map(l => (
            <Link key={l.text} href={l.href}>
              <a>{l.text}</a>
            </Link>
          ))}
          <Link href='https://github.com/infrahq/infra'>
            <a>
              <div className='rounded-full overflow-hidden bg-gradient-to-tr from-cyan-100 to-pink-300'>
                <button className='flex items-center m-px pl-2 pr-3 py-1.5 rounded-full bg-black hover:bg-gray-200'><img className='pr-2 h-5' src='/images/github.svg' /> Open in GitHub</button>
              </div>
            </a>
          </Link>
        </nav>
        <div className='flex md:hidden flex-row items-center space-x-12 text-base py-2 px-4 -mr-4' onClick={() => setOpen(!open)}>
          {open ? <XIcon className='w-6 h-6 text-zinc-200' /> : <MenuIcon className='w-6 h-6 text-zinc-200' />}
        </div>
      </div>
      <nav className={`flex flex-col w-full relative z-10 justify-around space-y-6 py-5 text-lg font-thin ${open ? '' : 'hidden'}`}>
        {links.map(l => (
          <Link key={l.text} href={l.href}>
            <a onClick={() => setOpen(false)} className='w-full py-2'>{l.text}</a>
          </Link>
        ))}
        <Link href='https://github.com/infrahq/infra'>
          <a>
            <div className='inline-flex rounded-full overflow-hidden bg-gradient-to-tr from-cyan-100 to-pink-300'>
              <button className='flex items-center m-px pl-3 pr-4 py-1.5 rounded-full bg-black hover:bg-gray-900'><img className='pr-2 h-5' src='/images/github.svg' /> Open in GitHub</button>
            </div>
          </a>
        </Link>
      </nav>
    </header>
  )
}
