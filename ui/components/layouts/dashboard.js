import Link from 'next/link'
import { Fragment, useState, forwardRef } from 'react'
import { useRouter } from 'next/router'
import useSWR, { useSWRConfig } from 'swr'
import { Dialog, Transition, Menu } from '@headlessui/react'
import {
  ChipIcon,
  UserGroupIcon,
  UserIcon,
  ViewGridIcon,
  XIcon,
  MenuIcon,
  CogIcon,
} from '@heroicons/react/outline'
import Avatar from 'boring-avatars'

import { useAdmin } from '../../lib/admin'

import AuthRequired from '../auth-required'
import { getAvatarName } from '../../lib/icons'

const NavLink = forwardRef(function NavLinkFunc(props, ref) {
  let { href, children, ...rest } = props
  return (
    <Link href={href}>
      <a ref={ref} {...rest}>
        {children}
      </a>
    </Link>
  )
})

function SidebarNav({ children, open, setOpen }) {
  return (
    <Transition.Root show={open} as={Fragment}>
      <Dialog as='div' className='relative z-40 md:hidden' onClose={setOpen}>
        <Transition.Child
          as={Fragment}
          enter='transition-opacity ease-linear duration-300'
          enterFrom='opacity-0'
          enterTo='opacity-100'
          leave='transition-opacity ease-linear duration-300'
          leaveFrom='opacity-100'
          leaveTo='opacity-0'
        >
          <div className='fixed inset-0 bg-gray-100/50 backdrop-blur-lg' />
        </Transition.Child>

        <div className='fixed inset-0 z-40 flex'>
          <Transition.Child
            as={Fragment}
            enter='transition ease-in-out duration-300 transform'
            enterFrom='-translate-x-full'
            enterTo='translate-x-0'
            leave='transition ease-in-out duration-300 transform'
            leaveFrom='translate-x-0'
            leaveTo='-translate-x-full'
          >
            <Dialog.Panel className='relative flex w-full max-w-[16rem] flex-1 flex-col bg-white px-6 pt-5 pb-4'>
              <Transition.Child
                as={Fragment}
                enter='ease-in-out duration-300'
                enterFrom='opacity-0'
                enterTo='opacity-100'
                leave='ease-in-out duration-300'
                leaveFrom='opacity-100'
                leaveTo='opacity-0'
              >
                <div className='absolute top-0 right-0 -mr-12 pt-2'>
                  <button
                    type='button'
                    className='justify-cente ml-1 flex h-10 w-10 items-center'
                    onClick={() => setOpen(false)}
                  >
                    <span className='sr-only'>Close sidebar</span>
                    <XIcon
                      className='h-6 w-6 text-gray-700 hover:text-gray-900'
                      aria-hidden='true'
                    />
                  </button>
                </div>
              </Transition.Child>
              {children}
            </Dialog.Panel>
          </Transition.Child>
          <div className='w-14 flex-shrink-0'></div>
        </div>
      </Dialog>
    </Transition.Root>
  )
}

function Layout({ children }) {
  const router = useRouter()

  const { data: auth } = useSWR('/api/users/self')
  const { admin, loading } = useAdmin()
  const { cache } = useSWRConfig()

  const [sidebarOpen, setSidebarOpen] = useState(false)

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
    {
      name: 'Infrastructure',
      href: '/destinations',
      icon: ChipIcon,
    },
    {
      name: 'Providers',
      href: '/providers',
      admin: true,
      icon: ViewGridIcon,
    },
    {
      name: 'Groups',
      href: '/groups',
      admin: true,
      icon: UserGroupIcon,
    },
    {
      name: 'Users',
      href: '/users',
      admin: true,
      icon: UserIcon,
    },
    {
      name: 'Settings',
      href: '/settings',
      admin: true,
      icon: CogIcon,
    },
  ]

  const subNavigation = [
    {
      name: 'Account',
      href: '/account',
      show: auth?.providerNames.includes('infra'),
    },
  ]

  for (const n of [...navigation]) {
    if (router.pathname.startsWith(n.href) && n.admin && !admin) {
      router.replace('/')
      return null
    }
  }

  function Nav() {
    return (
      <>
        <div className='mb-2 flex flex-shrink-0 select-none items-center px-3'>
          <Link href='/destinations'>
            <a>
              <img className='my-2 h-7' src='/logo.svg' alt='Infra' />
            </a>
          </Link>
        </div>
        <div className='mt-5 h-0 flex-1 overflow-y-auto'>
          <nav className='flex-1 space-y-1'>
            {navigation
              ?.filter(n => (n.admin ? admin : true))
              .map(item => (
                <Link key={item.name} href={item.href}>
                  <a
                    onClick={() => setSidebarOpen(false)}
                    className={`
                          ${
                            router.asPath.startsWith(item.href)
                              ? 'bg-gray-100/50 text-gray-800'
                              : 'bg-transparent text-gray-500/75 hover:text-gray-500'
                          }
                        group flex items-center rounded-md py-1.5 px-3 text-sm font-medium`}
                  >
                    <item.icon
                      className={`${
                        router.asPath.startsWith(item.href)
                          ? 'fill-blue-100 text-blue-500'
                          : 'fill-gray-50 text-gray-500/75 group-hover:text-gray-500'
                      }
                    mr-2 h-[18px] w-[18px] flex-shrink`}
                      aria-hidden='true'
                    />
                    {item.name}
                  </a>
                </Link>
              ))}
          </nav>
        </div>
      </>
    )
  }

  return (
    <div className='relative flex'>
      <>
        <SidebarNav open={sidebarOpen} setOpen={setSidebarOpen}>
          <Nav navigation={navigation} admin={admin} />
        </SidebarNav>
        <div className='sticky top-0 hidden h-screen w-48 flex-none flex-col border-r border-gray-100 px-3 pt-5 pb-4 md:flex lg:w-60'>
          <Nav navigation={navigation} admin={admin} />
        </div>
      </>

      {/* Main content */}
      <div className='mx-auto flex min-w-0 flex-1 flex-col'>
        <div className='sticky top-0 z-10 flex flex-shrink-0 border-b border-gray-100 bg-white/90 py-3 px-6 pl-2 backdrop-blur-lg md:py-2 md:px-6'>
          <button
            type='button'
            className='px-4 text-black focus:outline-none focus:ring-2 focus:ring-inset focus:ring-blue-500 md:hidden'
            onClick={() => setSidebarOpen(true)}
          >
            <span className='sr-only'>Open sidebar</span>
            <MenuIcon className='h-6 w-6' aria-hidden='true' />
          </button>
          <div className='flex flex-1 justify-end'>
            <div className='ml-4 flex items-center md:ml-6'>
              <Menu
                as='div'
                className='relative inline-block bg-white text-left'
              >
                <span className='sr-only'>Open current user menu</span>
                {/* <Menu.Button className='flex h-8 w-8 select-none items-center justify-center rounded-full bg-blue-500 text-white'>
                  <span className='text-center text-xs font-semibold capitalize leading-none'>
                    {auth?.name?.[0]}
                  </span>
                </Menu.Button> */}
                <Menu.Button>
                  <Avatar
                    size={30}
                    name={getAvatarName(auth?.name)}
                    variant='beam'
                    colors={[
                      '#343838',
                      '#005F6B',
                      '#008C9E',
                      '#00B4CC',
                      '#00DFFC',
                    ]}
                  />
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
                      <p className='text-xs text-gray-600'>Signed in as</p>
                      <p className='truncate text-sm font-semibold text-gray-900'>
                        {auth?.name}
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
                          Sign out
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

export default function Dashboard(props) {
  return (
    <AuthRequired>
      <Layout {...props} />
    </AuthRequired>
  )
}
