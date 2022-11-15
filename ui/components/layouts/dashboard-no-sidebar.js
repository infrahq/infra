import Link from 'next/link'
import { Fragment, forwardRef } from 'react'
import { useRouter } from 'next/router'
import { Transition, Menu } from '@headlessui/react'

import { useUser } from '../../lib/hooks'

const NavLink = forwardRef(function NavLinkFunc(props, ref) {
  let { href, children, ...rest } = props
  return (
    <Link href={href} ref={ref} {...rest}>
      {children}
    </Link>
  )
})

export default function DashboardNoSidebar({ children }) {
  const router = useRouter()

  const { user, loading, org, logout } = useUser({
    redirectTo:
      router.asPath === '/'
        ? '/login'
        : `/login?next=${encodeURIComponent(router.asPath)}`,
  })

  if (loading) {
    return null
  }

  const subNavigation = [
    {
      name: 'Account',
      href: '/account',
      show: user?.providerNames?.includes('infra'),
    },
  ]

  return (
    <div className='relative flex'>
      {/* Main content */}
      <div className='mx-auto flex min-w-0 flex-1 flex-col'>
        <div className='sticky top-0 z-10 flex flex-shrink-0 border-b border-gray-100 bg-white/90 py-3 px-6 pl-2 backdrop-blur-lg md:py-2 md:px-6'>
          <div className='flex flex-1 justify-end'>
            <div className='ml-4 flex items-center md:ml-6'>
              <Menu
                as='div'
                className='relative inline-block bg-white text-left'
              >
                <span className='sr-only'>Open current user menu</span>
                <Menu.Button className='flex h-8 w-8 select-none items-center justify-center rounded-full bg-blue-500 text-white'>
                  <span className='text-center text-xs font-semibold capitalize leading-none'>
                    {user?.name?.[0]}
                  </span>
                </Menu.Button>
                <Transition
                  as={Fragment}
                  enter='transition ease-out duration-100'
                  enterFrom='transform opacity-0 scale-95'
                  enterTo='transform opacity-100 scale-100'
                  leave='transition ease-in duration-75'
                  leaveFrom='transform opacity-100 scale-100'
                  leaveTo='transform opacity-0 scale-95'
                >
                  <Menu.Items className='absolute right-0 z-50 mt-2 w-56 origin-top-right divide-y divide-gray-100 rounded-md bg-white shadow-xl shadow-black/5 ring-1 ring-black ring-opacity-5 focus:outline-none'>
                    <div className='px-4 py-3'>
                      <p className='text-xs text-gray-400'>Logged in as</p>
                      <p className='mt-2 truncate text-sm font-semibold text-gray-900'>
                        {user?.name}
                      </p>
                      <p className='truncate text-sm text-gray-600'>
                        {org?.name}
                      </p>
                    </div>

                    {subNavigation?.filter(n => n.show).length > 0 && (
                      <div className='py-1'>
                        {subNavigation
                          ?.filter(n => n.show)
                          .map(item => (
                            <Menu.Item key={item.name}>
                              <NavLink href={item.href}>
                                <p className='block py-2 px-4 text-sm text-gray-700 hover:bg-gray-100'>
                                  {item.name}
                                </p>
                              </NavLink>
                            </Menu.Item>
                          ))}
                      </div>
                    )}
                    <div className='py-1'>
                      <Menu.Item>
                        <button
                          type='button'
                          onClick={() => logout()}
                          className='block w-full cursor-pointer py-2 px-4 text-left text-sm text-gray-700 hover:bg-gray-100'
                        >
                          Log out
                        </button>
                      </Menu.Item>
                    </div>
                  </Menu.Items>
                </Transition>
              </Menu>
            </div>
          </div>
        </div>

        <main className='mx-auto w-full max-w-6xl flex-1 px-6 '>
          {children}
        </main>
      </div>
    </div>
  )
}
