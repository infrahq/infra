import Link from 'next/link'
import { useRouter } from 'next/router'
import { useEffect, useState, useRef } from 'react'
import { ChevronRightIcon, ExternalLinkIcon } from '@heroicons/react/solid'

import Nav from './Nav'
import SignupForm from './SignupForm'

function Expandable({ expanded, children }) {
  const ref = useRef()
  const [height, setHeight] = useState('auto')

  useEffect(() => {
    setHeight(ref.current?.offsetHeight || 0)
  }, [])

  return (
    <div
      style={{ height: expanded ? height : 0 }}
      className='duration-250 overflow-y-hidden transition-[height]'
    >
      <div ref={ref}>{children}</div>
    </div>
  )
}

function Category({ href, title, empty, items }) {
  const router = useRouter()
  const [expanded, setExpanded] = useState(router.asPath.startsWith(href))

  useEffect(() => {
    setExpanded(router.asPath.startsWith(href))
  }, [router.asPath, href])

  return (
    <div>
      <div
        onClick={() => empty && setExpanded(!expanded)}
        className='relative flex cursor-pointer items-center py-0.5'
      >
        <div
          onClick={() => setExpanded(!expanded)}
          className='absolute -left-5 mb-0.5 pr-1'
        >
          <ChevronRightIcon
            className={`duration-250 inline h-4 w-4 text-gray-600 transition-transform ${
              expanded ? 'rotate-90' : 'rotate-0'
            }`}
          />
        </div>
        {empty ? (
          <span className='flex flex-1 select-none py-1.5 leading-none'>
            {title}
          </span>
        ) : (
          <Page href={href} title={title} />
        )}
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

function Page({ title, href }) {
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
      <a
        ref={ref}
        target={external ? '_blank' : ''}
        className={`flex flex-1 select-none py-1.5 leading-none ${
          active ? 'font-semibold text-white' : ''
        }`}
      >
        {title}{' '}
        {external && (
          <ExternalLinkIcon className='ml-1 h-3.5 w-3.5 text-zinc-400' />
        )}
      </a>
    </Link>
  )
}

function NavItem({ item }) {
  return item.items ? <Category {...item} /> : <Page {...item} />
}

function MobileNavItem({ item, depth }) {
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

  return <option value={item.href}>{item.title}</option>
}

export default function DocsLayout({
  children,
  items = [],
  headings = [],
  icon,
}) {
  const router = useRouter()
  const [id, setId] = useState('')
  const ref = useRef()

  const margin = 90

  useEffect(() => {
    function onScroll() {
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

    onScroll()
    window.addEventListener('scroll', onScroll, { passive: true })
    return () => window.removeEventListener('scroll', onScroll)
  }, [children, headings])

  return (
    <main className='flex w-full flex-1 flex-col justify-center'>
      <Nav docs />
      <div className='mx-auto flex w-full max-w-screen-2xl flex-col md:flex-row'>
        <ul className='fixed hidden max-h-[calc(100vh-4rem)] min-h-0 flex-none flex-col self-start overflow-y-auto py-10 px-8 text-lg tracking-[-0.02em] text-gray-300 md:flex md:w-56 md:flex-none lg:w-64'>
          {items.map(i => (
            <NavItem key={i.href} item={i} />
          ))}
        </ul>
        <select
          value={router.asPath}
          className='mx-6 rounded-lg border border-zinc-600 bg-transparent py-2 px-2 text-white md:hidden'
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
        <div className='min-w-0 flex-1 pl-0 md:pl-56 lg:pl-64'>
          <div className='relative my-8 mx-auto flex w-full min-w-0 flex-1 flex-col px-8 md:pl-0 lg:max-w-2xl lg:px-0 xl:max-w-3xl'>
            {icon && <img alt='icon' className='h-16 w-16' src={icon} />}
            <div className='prose-docs prose-md prose prose-invert w-full max-w-none break-words'>
              {children}
            </div>
            <hr className='my-12 border-zinc-800' />
            <div className='mx-auto mb-20 max-w-sm text-center'>
              <h1 className='my-6 text-xl font-bold tracking-tight'>
                Sign up for updates
              </h1>
              <SignupForm />
              <h2 className='my-2 text-sm text-gray-300'>
                You can unsubscribe at any time.
              </h2>
            </div>
          </div>
        </div>
        <aside className='text-md sticky top-20 hidden max-h-[calc(100vh-4rem)] min-h-0 flex-none self-start overflow-auto py-10 lg:block lg:w-64 lg:px-10 xl:w-72'>
          {headings.length > 0 && (
            <>
              <h2 className='text-md mb-2 font-normal text-white'>
                On this page
              </h2>
              {headings
                .filter(h => h.level <= 3)
                .map(h => (
                  <Link key={h.id} href={`#${h.id}`}>
                    <a
                      ref={h.id === id ? ref : null}
                      className={`block py-1 leading-tight text-zinc-400 hover:text-zinc-100 ${
                        h.id === id ? 'text-zinc-100' : ''
                      } ${h.level > 2 ? 'ml-3' : ''}`}
                    >
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
