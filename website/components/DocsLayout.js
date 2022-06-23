import Link from 'next/link'
import { useRouter } from 'next/router'
import { useEffect, useState, useRef } from 'react'
import { ChevronRightIcon, ExternalLinkIcon } from '@heroicons/react/solid'

import Nav from './Nav'
import SignupForm from './SignupForm'

function Expandable ({ expanded, children }) {
  const ref = useRef()
  const [height, setHeight] = useState('auto')

  useEffect(() => {
    setHeight(ref.current?.offsetHeight || 0)
  }, [])

  return (
    <div style={{ height: expanded ? height : 0 }} className='overflow-y-hidden duration-250 transition-[height]'>
      <div ref={ref}>
        {children}
      </div>
    </div>
  )
}

function Category ({ href, title, empty, items }) {
  const router = useRouter()
  const [expanded, setExpanded] = useState(router.asPath.startsWith(href))

  useEffect(() => {
    setExpanded(router.asPath.startsWith(href))
  }, [router.asPath, href])

  return (
    <div>
      <div onClick={() => empty && setExpanded(!expanded)} className='flex items-center cursor-pointer relative py-0.5'>
        <div onClick={() => setExpanded(!expanded)} className='pr-1 absolute -left-5 mb-0.5'>
          <ChevronRightIcon className={`text-gray-600 inline w-4 h-4 duration-250 transition-transform ${expanded ? 'rotate-90' : 'rotate-0'}`} />
        </div>
        {empty ? <span className='flex flex-1 py-1.5 leading-none select-none'>{title}</span> : <Page href={href} title={title} />}
      </div>
      <Expandable expanded={expanded}>
        <div className='pb-3'>
          {items?.map(i => (
            <div key={i.href} className='ml-3 flex'>
              <NavItem item={i} />
            </div>
          ))}
        </div>
      </Expandable>
    </div>
  )
}

function Page ({ title, href }) {
  const router = useRouter()
  const active = router.asPath.split('#')[0].split('?')[0] === href
  const external = href?.startsWith('https://')
  const ref = useRef()

  useEffect(() => {
    if (active) {
      if (ref.current?.scrollIntoViewIfNeeded) {
        ref.current?.scrollIntoViewIfNeeded()
      }
    }
  }, [active])

  return (
    <Link href={href} passHref={external}>
      <a ref={ref} target={external ? '_blank' : ''} className={`flex flex-1 py-1.5 leading-none select-none ${active ? 'text-white font-semibold' : ''}`}>{title} {external && <ExternalLinkIcon className='ml-1 text-zinc-400 w-3.5 h-3.5' />}</a>
    </Link>
  )
}

function NavItem ({ item }) {
  return item.items
    ? <Category {...item} />
    : <Page {...item} />
}

function MobileNavItem ({ item, depth }) {
  depth = depth || 0

  if (item.items) {
    return (
      <>
        {item.items?.map(i => (
          <MobileNavItem key={i.href} item={i} depth={depth + 1} />
        ))}
      </>
    )
  }

  return (
    <option value={item.href}>{item.title}</option>
  )
}

export default function DocsLayout ({ children, items = [], headings = [], icon }) {
  const router = useRouter()
  const [id, setId] = useState('')
  const ref = useRef()

  const margin = 90

  function onScroll (e) {
    let active = null

    for (const h of headings.filter(h => h.level <= 3)) {
      const element = document.getElementById(h.id)
      if (!element) {
        continue
      }

      if (element.getBoundingClientRect().top < margin) {
        active = element
      }
    }

    if (!active) {
      setId(headings[0]?.id || '')
      return
    }

    setId(active.id)

    if (ref && ref.current?.scrollIntoViewIfNeeded) {
      ref.current?.scrollIntoViewIfNeeded()
    }
  }

  useEffect(() => {
    onScroll()
    window.addEventListener('scroll', onScroll, { passive: true })
    return () => window.removeEventListener('scroll', onScroll)
  }, [children])

  return (
    <main className='flex-1 flex flex-col justify-center w-full'>
      <Nav docs />
      <div className='flex flex-col md:flex-row max-w-screen-2xl w-full mx-auto'>
        <ul className='flex-none fixed self-start min-h-0 max-h-[calc(100vh-4rem)] overflow-y-scroll md:w-56 lg:w-64 py-10 px-8 hidden md:flex md:flex-none flex-col text-lg tracking-[-0.02em] text-gray-300'>
          {items.map(i => (
            <NavItem key={i.href} item={i} />
          ))}
        </ul>
        <select
          value={router.asPath}
          className='md:hidden bg-transparent border text-white border-zinc-600 mx-6 py-2 px-2 rounded-lg'
          onChange={e => router.push(e.target.value)}
        >
          {items.map(i => (
            <optgroup key={i.href} label={i.title}>
              {i.items?.map(d => (
                <MobileNavItem key={d.href} item={d} />
              ))}
            </optgroup>
          ))}
        </select>
        <div className='flex-1 min-w-0 pl-0 md:pl-56 lg:pl-64'>
          <div className='relative flex-1 flex flex-col px-8 md:pl-0 lg:px-0 my-8 min-w-0 lg:max-w-2xl xl:max-w-3xl w-full mx-auto'>
            {icon && (<img className='w-16 h-16' src={icon} />)}
            <div className='w-full max-w-none prose prose-docs prose-md prose-invert break-words'>
              {children}
            </div>
            <hr className='my-12 border-zinc-800' />
            <div className='max-w-sm text-center mx-auto mb-20'>
              <h1 className='text-xl font-bold tracking-tight my-6'>Sign up for updates</h1>
              <SignupForm />
              <h2 className='text-sm my-2 text-gray-300'>You can unsubscribe at any time.</h2>
            </div>
          </div>
        </div>
        <aside className='sticky top-20 lg:w-64 xl:w-72 min-h-0 max-h-[calc(100vh-4rem)] flex-none self-start hidden lg:block py-10 lg:px-10 text-md overflow-scroll'>
          {headings.length > 0 && (
            <>
              <h2 className='font-normal text-white mb-2 text-md'>On this page</h2>
              {headings.filter(h => h.level <= 3).map(h => (
                <Link key={h.id} href={`#${h.id}`}>
                  <a ref={h.id === id ? ref : null} className={`block py-1 leading-tight text-zinc-400 hover:text-zinc-100 ${h.id === id ? 'text-zinc-100' : ''} ${h.level > 2 ? 'ml-3' : ''}`}>
                    {h.title}
                  </a>
                </Link>
              ))}
            </>
          )}
        </aside>
      </div>
    </main>
  )
}
