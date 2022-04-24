import Link from 'next/link'
import { useRouter } from 'next/router'
import useSWR, { useSWRConfig } from 'swr'
import classNames from 'classnames'

import { useAdmin } from '../lib/admin'

export default function ({ children }) {
  const router = useRouter()
  const { data: auth } = useSWR('/v1/introspect')
  const { data: version } = useSWR('/v1/version')
  const { admin, loading } = useAdmin()
  const { mutate } = useSWRConfig()

  if (loading) {
    return null
  }

  async function logout () {
    fetch('/v1/logout', {
      method: 'POST'
    })
    mutate('/v1/introspect', undefined)
    router.replace('/login')
  }

  const navigation = [
    { name: 'Clusters', href: '/destinations', icon: '/infrastructure.svg' },
    { name: 'Identity Providers', href: '/providers', icon: '/providers.svg', admin: true }
  ]

  const subNavigation = [
    { name: 'Settings', href: '/settings', icon: '/settings.svg', admin: true},
  ]

  // redirect non-admin routes if user isn't admin
  for (const n of [...navigation, ...subNavigation]) {
    if (router.asPath.startsWith(n.href) && n.admin && !admin) {
      router.replace('/')
      return null
    }
  }

  return (
    <div className='flex h-full relative'>
      <nav className='flex-none flex w-64 lg:w-72 flex-col inset-y-0 px-2 overflow-y-auto'>
        <div className='flex-shrink-0 flex items-center my-12 lg:my-18 px-6 select-none'>
          <Link href='/'>
            <a>
              <img
                className='h-[18px] w-auto'
                src='infra.svg'
                alt='Infra'
              />
            </a>
          </Link>
        </div>
        <div className='flex-1 space-y-1.5 px-3 select-none'>
          {navigation.map(n =>
            <Link key={n.name} href={n.href}>
              <a
                href={n.href}
                className={`
                  ${router.asPath.startsWith(n.href) ? 'bg-purple-200/10 text-white' : 'text-gray-500 hover:bg-purple-200/5 hover:text-gray-300'}
                  rounded-lg py-2 px-3 flex items-center text-sm font-medium transition-colors duration-100
                  ${n.admin && !admin ? 'opacity-30 pointer-events-none' : ''}
                `}
              >
                <img
                  src={n.icon}
                  className={classNames(
                    router.asPath.startsWith(n.href) ? '' : 'opacity-30',
                    'mr-3 flex-shrink-0 h-5 w-5'
                  )}
                />
                {n.name}
              </a>
            </Link>
          )}
        </div>
        <div className='relative group mx-2 my-5 h-16 hover:h-[178px] hover:bg-purple-100/5 transition-height transition-size px-4 duration-300 ease-in-out rounded-xl overflow-hidden'>
          <div className='flex items-center space-x-4 mt-4 mb-2'>
            <div className='bg-purple-100/10 flex-none flex items-center justify-center w-9 h-9 py-1.5 rounded-lg capitalize font-bold select-none'>{auth?.name?.[0]}</div>
            <div>
              <div className='text-gray-300 text-sm font-medium overflow-hidden overflow-ellipsis leading-none'>{auth?.name}</div>
              {admin && <div className='text-gray-400 text-xs leading-none my-1 capitalize'>Admin</div>}
            </div>
          </div>
          <div className='w-full px-2 py-1 items-center opacity-0 group-hover:opacity-100 transition-opacity duration-300 select-none text-sm'>
            <div onClick={() => logout()} className='w-full flex items-center opacity-50 hover:opacity-75 py-2 cursor-pointer'>
              <img src='/signout.svg' className='opacity-50 group-hover:opacity-75 h-3 mr-3' />
              <div className='text-purple-50/40 group-hover:text-purple-50'>Logout</div>
            </div>
            {subNavigation.map(s => (
              <Link key={s.name} href={s.href}>
                <a className={`w-full flex -ml-1 opacity-50 hover:opacity-75 py-2 ${s.admin && !admin ? 'pointer-events-none opacity-20' : ''}`}>
                  <img src={s.icon} className='opacity-50 group-hover:opacity-75 mr-3 w-5 h-5' />
                  <div className='text-purple-50/40 group-hover:text-purple-50'>{s.name}</div>
                </a>
              </Link>
            ))}
          </div>
          <div className='px-2 pt-1 pb-3 text-xs text-purple-50/30'>
            Infra version {version?.version}
          </div>
        </div>
      </nav>
      <main className='w-full mx-auto xl:max-w-4xl 2xl:max-w-5xl overflow-x-hidden overflow-y-scroll'>
        {children}
      </main>
    </div>
  )
}
