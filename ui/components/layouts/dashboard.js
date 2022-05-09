import Link from 'next/link'
import { useRouter } from 'next/router'
import useSWR, { useSWRConfig } from 'swr'

import { useAdmin } from '../../lib/admin'

export default function ({ children }) {
  const router = useRouter()
  const { data: auth } = useSWR('/v1/identities/self')
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
    await mutate('/v1/identities/self', async () => undefined)
    router.replace('/login')
  }

  const navigation = [
    { name: 'Infrastructure', href: '/destinations', icon: '/destinations.svg' },
    { name: 'Providers', href: '/providers', icon: '/providers.svg', admin: true }
  ]

  const subNavigation = [
    { name: 'Settings', href: '/settings', admin: true }
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
      <nav className='flex-none flex w-1/6 flex-col inset-y-0 px-2 overflow-y-auto'>
        <div className='flex-shrink-0 flex items-center mt-6 mb-10 lg:my-18 px-6 select-none'>
          <Link href='/'>
            <a>
              <img
                className='h-[13px] w-[36px]'
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
                  ${router.asPath.startsWith(n.href) ? 'text-white' : 'text-gray-400'}
                  rounded-lg py-2 px-3 flex items-center text-title transition-colors duration-100 
                  ${n.admin && !admin ? 'opacity-30 pointer-events-none' : ''}
                `}
              >
                <img
                  src={n.icon}
                  className={`${router.asPath.startsWith(n.href) ? '' : 'opacity-40'} mr-3 flex-shrink-0 h-3.5 w-3.5`}
                />
                  {n.name}
              </a>
            </Link>
          )}
        </div>
        <div className='relative group mx-2 my-5 px-6 pb-20 h-16 hover:h-40 transition-all duration-300 ease-in-out rounded-xl overflow-hidden bg-transparent hover:bg-gray-900 shadow hover:shadow-lg'>
          <div className='flex items-center space-x-2 mt-4 mb-2'>
            <div className='bg-gradient-to-tr from-indigo-300/40 to-pink-100/40 rounded-[4px] p-px'>
              <div className='bg-black flex-none flex items-center justify-center w-8 h-8 rounded-[4px]'>
                <div className='bg-gradient-to-tr from-indigo-300 to-pink-100 rounded-sm p-px'>
                  <div className='bg-black flex-none flex justify-center items-center w-6 h-6 pb-0.5 text-subtitle font-bold select-none rounded-sm'>
                    {auth?.name?.[0]}
                  </div>
                </div>
              </div>
            </div>
            <div className='text-gray-400 hover:text-white text-title leading-none truncate'>{auth?.name}</div>
          </div>
          <div className='w-full pl-11 pr-2 items-center opacity-0 group-hover:opacity-100 transition-opacity duration-300 select-none text-sm'>
            {subNavigation.map(s => (
              <Link key={s.name} href={s.href}>
                <a className={`w-full flex py-2 ${s.admin && !admin ? 'pointer-events-none opacity-20' : ''}`}>
                  <div className='text-gray-400 text-title hover:text-white'>{s.name}</div>
                </a>
              </Link>
            ))}
            <div onClick={() => logout()} className='w-full flex items-center py-2 cursor-pointer'>
              <div className='text-gray-400 text-title hover:text-white'>Sign Out</div>
            </div>
            <div className='pt-2 pb-4 text-note text-gray-400/40'>
              Infra version {version?.version}
            </div>
          </div>
        </div>
      </nav>
      <main className='w-full overflow-x-hidden overflow-y-scroll'>
        <div className='mx-10'>
          {children}
        </div>
      </main>
    </div>
  )
}
