import Link from 'next/link'
import { useRouter } from 'next/router'
import { useEffect, useState, useRef } from 'react'
import {
  ArrowTopRightOnSquareIcon,
  ChevronRightIcon,
} from '@heroicons/react/24/outline'

function Expandable({ expanded, children }) {
  const ref = useRef()
  const [height, setHeight] = useState('auto')

  const observer = useRef(
    typeof ResizeObserver === 'undefined'
      ? null
      : new ResizeObserver(entries => {
          const { height } = entries[0].contentRect
          setHeight(height)
        })
  )

  useEffect(() => {
    const obs = observer.current
    const el = ref.current
    if (el) {
      obs.observe(el)
    }

    return () => {
      obs.unobserve(el)
    }
  }, [ref, observer])

  return (
    <div
      style={{ height: expanded ? height : 0 }}
      className='overflow-y-hidden'
    >
      <div ref={ref}>{children}</div>
    </div>
  )
}

function Category({ href, title, items, empty }) {
  const router = useRouter()
  const [expanded, setExpanded] = useState(router.asPath.startsWith(href))

  useEffect(() => {
    setExpanded(router.asPath.startsWith(href))
  }, [router.asPath, href])

  return (
    <ol>
      <div
        onClick={() => setExpanded(!expanded)}
        className='relative flex cursor-pointer items-center py-0.5 font-medium text-gray-700'
      >
        {items && (
          <span className='flex flex-1 select-none items-center leading-4'>
            <ChevronRightIcon
              className={`relative mb-px mr-1 h-3 stroke-[3px] text-gray-400 transition-transform ${
                expanded ? 'rotate-90' : 'rotate-0'
              }`}
            />{' '}
            {empty ? (
              <span className='py-2'>{title}</span>
            ) : (
              <Page href={href} items={items} title={title} />
            )}
          </span>
        )}
      </div>
      <Expandable expanded={expanded}>
        <div className='ml-2 pb-3'>
          {items?.map(i => (
            <div key={i.href} className='flex'>
              <NavItem item={i} />
            </div>
          ))}
        </div>
      </Expandable>
    </ol>
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
        className={`ml-4 flex flex-1 select-none py-1.5 leading-none ${
          active ? 'font-medium text-blue-600' : ''
        }`}
      >
        {title}{' '}
        {external && <ArrowTopRightOnSquareIcon className='ml-1 h-3.5 w-3.5' />}
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

export default function DocsLayout({ children, items = [], headings = [] }) {
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
    }

    onScroll()
    window.addEventListener('scroll', onScroll, { passive: true })
    return () => window.removeEventListener('scroll', onScroll)
  }, [children, headings])

  return (
    <div className='px-6'>
      <div className='mx-auto flex h-full w-full max-w-7xl flex-col md:flex-row'>
        <ul className='sticky top-20 -ml-4 hidden min-h-0 flex-none flex-col self-start overflow-y-auto py-8 pr-6 text-sm text-zinc-600 md:flex md:w-48 md:flex-none xl:w-56'>
          {items.map(i => (
            <NavItem key={i.href} item={i} />
          ))}
        </ul>
        <select
          value={router.asPath}
          className='my-3 rounded-lg border border-gray-300 bg-transparent py-2 px-2 md:hidden'
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
        <div className='prose-docs prose-md prose relative mb-32 flex w-full w-full min-w-0 max-w-none flex-1 flex-col break-words md:pl-0'>
          {children}
        </div>
        <aside className='left-full ml-6 hidden w-48 lg:ml-8 lg:block xl:ml-12'>
          <div className='sticky top-32 mb-32 overflow-y-auto text-xs text-zinc-500'>
            {headings.length > 0 && (
              <>
                <h2 className='mb-2 font-semibold tracking-tight text-black'>
                  On this page
                </h2>
                {headings
                  .filter(h => h.level <= 3)
                  .map(h => (
                    <Link key={h.id} href={`#${h.id}`}>
                      <a
                        ref={h.id === id ? ref : null}
                        className={`block py-1 leading-tight ${
                          h.id === id ? 'text-black' : ''
                        } ${h.level > 2 ? 'ml-3' : ''}`}
                      >
                        {h.title}
                      </a>
                    </Link>
                  ))}
              </>
            )}
          </div>
        </aside>
      </div>
    </div>
  )
}
