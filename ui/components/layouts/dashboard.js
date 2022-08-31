import Link from 'next/link'
import { useRouter } from 'next/router'
import useSWR, { useSWRConfig } from 'swr'

import { useAdmin } from '../../lib/admin'
import AuthRequired from '../auth-required'

function Layout({ children }) {
  const router = useRouter()
  const { data: auth } = useSWR('/api/users/self')
  const { data: version } = useSWR('/api/version')
  const { admin, loading } = useAdmin()
  const { cache } = useSWRConfig()

  const accessToSettingsPage = admin || auth?.providerNames?.includes('infra')

  if (loading) {
    return null
  }

  async function logout() {
    await fetch('/api/logout', {
      method: 'POST',
    })
    cache.clear()
    router.replace('/login')
  }

  const navigation = [
    { name: 'Clusters', href: '/destinations', icon: '/destinations.svg' },
    {
      name: 'Providers',
      href: '/providers',
      icon: '/providers.svg',
      admin: true,
    },
    { name: 'Groups', href: '/groups', icon: '/groups.svg', admin: true },
    { name: 'Users', href: '/users', icon: '/users.svg', admin: true },
  ]

  const subNavigation = [
    { name: 'Settings', href: '/settings', admin: accessToSettingsPage },
  ]

  // redirect non-admin routes if user isn't admin
  if (router.pathname.startsWith('/settings') && !accessToSettingsPage) {
    router.replace('/')
    return null
  }

  for (const n of [...navigation]) {
    if (router.pathname.startsWith(n.href) && n.admin && !admin) {
      router.replace('/')
      return null
    }
  }

  return (
    <div className='flex h-full min-w-[800px]'>
      <nav className='flex w-48 flex-none flex-col xl:w-56'>
        <div className='lg:my-18 mt-6 mb-10 flex flex-shrink-0 select-none items-center px-5'>
          <Link href='/'>
            <a>
              <img className='h-[15px]' src='infra.svg' alt='Infra' />
            </a>
          </Link>
        </div>
        <div className='flex-1 select-none space-y-1 px-5'>
          {navigation
            ?.filter(n => (n.admin ? admin : true))
            .map(n => (
              <Link key={n.name} href={n.href}>
                <a
                  href={n.href}
                  className={`
                    ${
                      router.asPath.startsWith(n.href)
                        ? 'text-white'
                        : 'text-gray-400'
                    }
                    flex items-center rounded-lg py-2 text-xs leading-none transition-colors duration-100
                  `}
                >
                  <img
                    alt={n?.name?.toLowerCase()}
                    src={n.icon}
                    className={`
                      ${router.asPath.startsWith(n.href) ? '' : 'opacity-40'}
                      mr-3 h-[18px] w-[18px] flex-shrink-0
                    `}
                  />
                  {n.name}
                </a>
              </Link>
            ))}
        </div>
        <div className='group mx-2 mb-2 flex h-12 overflow-hidden rounded-xl bg-transparent p-2.5 pb-1 transition-all duration-300 ease-in-out hover:h-[132px] hover:bg-gray-900'>
          <div className='flex h-[23px] w-[23px] select-none items-center justify-center rounded-md border border-gray-800'>
            <span className='text-center text-3xs font-normal leading-none text-gray-400'>
              {auth?.name?.[0]}
            </span>
          </div>
          <div className='ml-1 min-w-0 flex-1 select-none px-2'>
            <div
              title={auth?.name}
              className='mt-[5px] mb-2 truncate pb-px text-2xs leading-none text-gray-400 transition-colors duration-300 group-hover:text-white'
            >
              {auth?.name}
            </div>
            <div className='opacity-0 transition-opacity duration-300 group-hover:opacity-100'>
              {subNavigation.map(s => (
                <Link key={s.name} href={s.href}>
                  <a
                    className={`flex w-full py-1.5 text-xs text-gray-400 hover:text-white ${
                      s.admin ? '' : 'pointer-events-none opacity-20'
                    }`}
                  >
                    {s.name}
                  </a>
                </Link>
              ))}
              <button
                onClick={() => logout()}
                className='w-full cursor-pointer py-1.5 text-left text-xs text-gray-400 hover:text-white'
              >
                Sign Out
              </button>
              <div className='mt-2 text-3xs leading-none text-violet-50/40'>
                version{' '}
                <span className='select-text font-mono'>
                  {version?.version}
                </span>
              </div>
            </div>
          </div>
        </div>
      </nav>
      <main className='h-full min-w-0 flex-1'>{children}</main>
    </div>
  )
}

export default function Dashboard(props) {
  return (
    <AuthRequired>
      <Layout {...props} />
    </AuthRequired>
  )
}
