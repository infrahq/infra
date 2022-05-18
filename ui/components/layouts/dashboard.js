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
      <nav className='flex-none flex w-56 flex-col inset-y-0 overflow-y-auto'>
        <div className='flex-shrink-0 flex items-center mt-6 mb-10 lg:my-18 px-5 select-none'>
          <Link href='/'>
            <a>
              <img
                className='h-[15px]'
                src='infra.svg'
                alt='Infra'
              />
            </a>
          </Link>
        </div>
        <div className='flex-1 space-y-1 px-5 select-none'>
          {navigation.map(n =>
            <Link key={n.name} href={n.href}>
              <a
                href={n.href}
                className={`
                  ${router.asPath.startsWith(n.href) ? 'text-white' : 'text-gray-400'}
                  rounded-lg py-2 flex items-center text-[13px] leading-none transition-colors duration-100
                  ${n.admin && !admin ? 'opacity-30 pointer-events-none' : ''}
                `}
              >
                <img
                  src={n.icon}
                  className={`${router.asPath.startsWith(n.href) ? '' : 'opacity-40'} mr-3 flex-shrink-0 h-[18px] w-[18px]`}
                />
                {n.name}
              </a>
            </Link>
          )}
        </div>
        <div className='relative flex group mx-2 mb-2 p-2.5 pb-1 h-12 hover:h-[132px] transition-all duration-300 ease-in-out rounded-xl overflow-hidden bg-transparent hover:bg-gray-900 shadow hover:shadow-lg'>
          <div className='flex flex-none self-start items-stretch bg-gradient-to-tr from-indigo-300/40 to-pink-100/40 rounded-[4px] w-[23px] h-[23px]'>
            <div className='flex-1 bg-gradient-to-tr border-2 border-black from-indigo-300/40 to-pink-100/40 rounded-[4px] p-px m-px'>
              <div className='bg-black w-full h-full flex justify-center items-center text-[12px] leading-none font-normal mb-px select-none rounded-sm'>
                {auth?.name?.[0]}
              </div>
            </div>
          </div>
          <div className='w-full ml-1 px-2 items-center select-none'>
            <div className='text-gray-400 group-hover:text-white transition-colors duration-300 mt-1 mb-2 leding-none truncate text-xs pb-px'>{auth?.name}</div>
            <nav className='opacity-0 group-hover:opacity-100 transition-opacity duration-300'>
              {subNavigation.map(s => (
                <Link key={s.name} href={s.href}>
                  <a className={`w-full flex py-1.5 text-[13px] ${s.admin && !admin ? 'pointer-events-none opacity-20' : ''}`}>
                    <div className='text-gray-400 hover:text-white'>{s.name}</div>
                  </a>
                </Link>
              ))}
              <div onClick={() => logout()} className='w-full flex items-center py-1.5 cursor-pointer'>
                <div className='text-gray-400 text-[13px] hover:text-white'>Sign Out</div>
              </div>
              <div className='text-[10px] mt-2 leading-none text-gray-300/50'>
                Infra version {version?.version}
              </div>
            </nav>
          </div>
        </div>
      </nav>
      <main className='w-full overflow-x-hidden overflow-y-scroll'>
        <div className='mx-6'>
          {children}
        </div>
      </main>
    </div>
  )
}
