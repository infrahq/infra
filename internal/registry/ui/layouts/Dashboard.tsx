import { Fragment, useState } from 'react'
import { Menu, Dialog, Transition } from '@headlessui/react'
import { MenuIcon, XIcon, CogIcon, ViewGridIcon, DownloadIcon } from '@heroicons/react/outline'
import classnames from 'classnames'
import Link from 'next/link'
import { useRouter } from 'next/router'
import { useCookies } from 'react-cookie'
import { useQuery } from 'react-query'

import { useRedirectToLoginOnUnauthorized } from '../util/redirect'
import { V1 } from '../gen/v1.pb'

export default function Layout ({ children }: { children: JSX.Element[] | JSX.Element }): JSX.Element {
    const [cookies] = useCookies(['login'])
    const [sidebarOpen, setSidebarOpen] = useState(false)
    const router = useRouter()

  // Note: we don't use this information yet (we should use an alternative method that fetches user email, etc)
  // but it's used to verify auth whenever the window is opened or brought to the foreground to automatically
  // log users out
  const { error } = useQuery(
    'status',
    () => V1.ListUsers({}),
    {
      refetchInterval: 5000,
    }
  )
  useRedirectToLoginOnUnauthorized(error)

    if (process.browser && !cookies.login) {
        router.replace("/login")
        return <></>
    }

  const navigation = [
    { name: 'Infrastructure', href: '/infrastructure', icon: ViewGridIcon },
    { name: 'Settings', href: '/settings', icon: CogIcon },
  ]
  
  const userNavigation = [
    {
      name: 'Logout',
      href: "#",
      onClick: async () => {
        await V1.Logout({})
        router.replace("/login")
      }
    },
  ]

  return (
    <div className="h-screen bg-gray-50 overflow-hidden flex">
      {/* Mobile sidebar */}
      <Transition.Root show={sidebarOpen} as={Fragment}>
        <Dialog
          as="div"
          static
          className="fixed inset-0 z-40 flex md:hidden"
          open={sidebarOpen}
          onClose={setSidebarOpen}
        >
          <Transition.Child
            as={Fragment}
            enter="transition-opacity ease-linear duration-300"
            enterFrom="opacity-0"
            enterTo="opacity-100"
            leave="transition-opacity ease-linear duration-300"
            leaveFrom="opacity-100"
            leaveTo="opacity-0"
          >
            <Dialog.Overlay className="fixed inset-0 bg-gray-600 bg-opacity-75" />
          </Transition.Child>
          <Transition.Child
            as={Fragment}
            enter="transition ease-in-out duration-300 transform"
            enterFrom="-translate-x-full"
            enterTo="translate-x-0"
            leave="transition ease-in-out duration-300 transform"
            leaveFrom="translate-x-0"
            leaveTo="-translate-x-full"
          >
            <div className="relative max-w-xs w-full bg-white pt-8 pb-4 flex-1 flex flex-col">
              <Transition.Child
                as={Fragment}
                enter="ease-in-out duration-300"
                enterFrom="opacity-0"
                enterTo="opacity-100"
                leave="ease-in-out duration-300"
                leaveFrom="opacity-100"
                leaveTo="opacity-0"
              >
                <div className="absolute top-0 right-0 -mr-12 pt-2">
                  <button
                    className="ml-1 flex items-center justify-center h-10 w-10 rounded-full focus:outline-none focus:ring-2 focus:ring-inset focus:ring-white"
                    onClick={() => setSidebarOpen(false)}
                  >
                    <span className="sr-only">Close sidebar</span>
                    <XIcon className="h-6 w-6 text-white" aria-hidden="true" />
                  </button>
                </div>
              </Transition.Child>
              <div className="flex-shrink-0 flex justify-center items-center select-none">
                <img
                  className="h-7 w-auto"
                  src="/combo.svg"
                  alt="Logo"
                />
              </div>
              <div className="mt-8 flex-1 h-0 overflow-y-auto">
                <nav className="px-4 space-y-1">
                  {navigation.map((item) => (
                    <Link key={item.name} href={item.href}>
                      <a
                        key={item.name}
                        className={classnames(
                          router.asPath.startsWith(item.href) ? 'bg-blue-50 text-blue-600' : 'text-gray-600 hover:bg-gray-50 hover:text-gray-900',
                          'group rounded-md py-2 px-2 flex items-center text-base font-medium'
                        )}
                      >
                        <item.icon
                          className={classnames(
                            router.asPath.startsWith(item.href) ? 'text-blue-600' : 'text-gray-500 group-hover:text-gray-900',
                            'mr-2 flex-shrink-0 h-5 w-5'
                          )}
                          aria-hidden="true"
                        />
                        {item.name}
                      </a>
                    </Link>
                  ))}
                </nav>
              </div>
            </div>
          </Transition.Child>
          <div className="flex-shrink-0 w-14">{/* Dummy element to force sidebar to shrink to fit close icon */}</div>
        </Dialog>
      </Transition.Root>
  
      {/* Desktop sidebar */}
      <div className="hidden md:flex md:flex-shrink-0">
        <div className="w-56 flex flex-col bg-white">
          <div className="border-r border-gray-100 pt-8 pb-4 flex flex-col flex-grow overflow-y-auto">
            <div className="flex-shrink-0 px-4 flex items-center self-center">
              <img
                className="h-6 w-auto"
                src="/combo.svg"
                alt="Logo"
              />
            </div>
            <div className="flex-grow mt-10 flex flex-col">
              <nav className="flex-1 px-4 space-y-1">
              {navigation.map(item => (
                <Link key={item.name} href={item.href}>
                  <a
                    key={item.name}
                    className={classnames(
                      router.asPath.startsWith(item.href) ? 'bg-blue-100 text-blue-700' : 'text-gray-500 hover:bg-gray-50 hover:text-gray-700',
                      'group rounded-lg py-1.5 px-2 flex items-center text-sm font-semibold'
                    )}
                  >
                    <item.icon
                      className={classnames(
                        router.asPath.startsWith(item.href) ? 'text-blue-600' : 'text-gray-500 group-hover:text-gray-700',
                        'mr-1 flex-shrink-0 h-4 w-4'
                      )}
                      aria-hidden="true"
                    />
                    {item.name}
                  </a>
                </Link>
              ))}
              </nav>
            </div>
          </div>
        </div>
      </div>

      {/* Header */}
      <div className="flex-1 max-w-5xl w-0 flex flex-col md:px-8">
        <header className="relative z-10 flex-shrink-0 h-16 border-b border-gray-200 flex bg-white md:border-transparent md:bg-transparent">
          <button
            className="border-r border-gray-200 px-4 text-gray-500 focus:outline-none focus:ring-2 focus:ring-inset focus:ring-blue-600 md:hidden"
            onClick={() => setSidebarOpen(true)}
          >
            <span className="sr-only">Open sidebar</span>
            <MenuIcon className="h-6 w-6" aria-hidden="true" />
          </button>
          <div className="flex-1 flex justify-end">
            <div className="ml-4 flex items-center md:ml-6 px-4 md:px-0">
              <Menu as="div" className="ml-3 relative">
              {({ open }) => (
                <>
                  <div className="flex items-center">
                    <Menu.Button className="max-w-xs flex items-center text-sm rounded-full focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-600">
                      <span className="sr-only">Open user menu</span>
                      <span className="inline-block h-8 w-8 rounded-full overflow-hidden bg-gray-100 md:bg-gray-200 md:border">
                        <svg className="h-full w-full text-gray-300 md:text-gray-400" fill="currentColor" viewBox="0 0 24 24">
                          <path d="M24 20.993V24H0v-2.996A14.977 14.977 0 0112.004 15c4.904 0 9.26 2.354 11.996 5.993zM16.002 8.999a4 4 0 11-8 0 4 4 0 018 0z" />
                        </svg>
                      </span>
                    </Menu.Button>
                  </div>
                  <Transition
                    show={open}
                    as={Fragment}
                    enter="transition ease-out duration-100"
                    enterFrom="transform opacity-0 scale-95"
                    enterTo="transform opacity-100 scale-100"
                    leave="transition ease-in duration-75"
                    leaveFrom="transform opacity-100 scale-100"
                    leaveTo="transform opacity-0 scale-95"
                  >
                    <Menu.Items
                    static
                    className="origin-top-right absolute right-0 mt-2 w-48 rounded-md shadow-lg bg-white ring-1 ring-black ring-opacity-5 py-1 focus:outline-none"
                    >
                      {userNavigation.map((item) => (
                        <Menu.Item key={item.name}>
                          {({ active }) => (
                            <Link key={item.name} href={item.href}>
                              <a
                                onClick={item.onClick}
                                className={classnames(
                                  active ? 'bg-gray-100' : '',
                                  'block py-2 px-4 text-sm text-gray-700'
                                )}
                              >
                                {item.name}
                              </a>
                            </Link>
                          )}
                        </Menu.Item>
                      ))}
                    </Menu.Items>
                  </Transition>
                </>
              )}
              </Menu>
            </div>
          </div>
        </header>

        {/* Main content */}
        <main className="flex-1 relative focus:outline-none px-4 md:px-0">
          {children}
        </main>
      </div>
    </div>
  )
}
