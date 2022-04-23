import Link from 'next/link'
import { useRouter } from 'next/router'
import useSWR, { useSWRConfig } from 'swr'
import classNames from 'classnames'

const navigation = [
  { name: 'Clusters', href: '/destinations', icon: '/infrastructure.svg' },
  { name: 'Identity Providers', href: '/providers', icon: '/providers.svg' }
]

export default function ({ children }) {
  const router = useRouter()
  const { data: auth } = useSWR('/v1/introspect')
  const { mutate } = useSWRConfig()

  async function logout () {
    fetch('/v1/logout', {
      method: 'POST'
    })
    mutate('/v1/introspect', undefined)
    router.replace('/')
  }

  return (
    <div className='flex h-full relative'>
      <nav className='flex-none flex w-64 lg:w-72 flex-col inset-y-0 px-2 overflow-y-auto'>
        <div className='flex-shrink-0 flex items-center my-12 lg:my-18 px-6'>
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
          {navigation.map(item => (
            <Link key={item.name} href={item.href}>
              <a
                href={item.href}
                className={classNames(
                  router.asPath.startsWith(item.href) ? 'bg-purple-200/10 text-white' : 'text-gray-500 hover:bg-purple-200/5',
                  'group rounded-md py-2 px-3 flex items-center text-sm font-medium transition-colors duration-200'
                )}
              >
                <img
                  src={item.icon}
                  className={classNames(
                    router.asPath.startsWith(item.href) ? '' : 'opacity-50',
                    'mr-3 flex-shrink-0 h-5 w-5'
                  )}
                  aria-hidden='true'
                />
                {item.name}
              </a>
            </Link>
          ))}
        </div>
        <div className='relative group mx-2 my-5 h-16 hover:h-40 hover:bg-purple-100/5 transition-height transition-size px-4 duration-300 ease-in-out rounded-xl overflow-hidden'>
          <div className='flex items-center space-x-4 my-4'>
            <div className='bg-purple-100/10 flex-none flex items-center justify-center w-9 h-9 py-1.5 rounded-lg capitalize font-bold select-none'>{auth?.name?.[0]}</div>
            <div className='text-gray-300 text-sm font-medium overflow-hidden overflow-ellipsis'>{auth?.name}</div>
          </div>
          <div className='absolute w-full px-2 items-center opacity-0 group-hover:opacity-100 transition-opacity duration-300 text-sm'>
            <Link href='/settings'>
              <a>
                <div className='w-full flex -ml-1 opacity-50 hover:opacity-75 py-2'>
                  <img src='/settings.svg' className='opacity-50 group-hover:opacity-75 mr-3 w-5 h-5' />
                  <div className='text-purple-50/40 group-hover:text-purple-50'>Settings</div>
                </div>
              </a>
            </Link>
            <div onClick={() => logout()} className='w-full flex items-center opacity-50 hover:opacity-75 py-2 cursor-pointer'>
              <img src='/signout.svg' className='opacity-50 group-hover:opacity-75 h-3 mr-3' />
              <div className='text-purple-50/40 group-hover:text-purple-50'>Logout</div>
            </div>
          </div>
        </div>
      </nav>
      <main className='w-full mx-auto xl:max-w-4xl 2xl:max-w-5xl overflow-x-hidden overflow-y-scroll'>
        {children}
      </main>
    </div>
  )
}
