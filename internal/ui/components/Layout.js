import { Fragment, useState } from 'react'
import { useCookies } from 'react-cookie'
import { Dialog, Transition } from '@headlessui/react'
import {
  MenuIcon,
  UsersIcon,
  XIcon,
  ViewGridIcon,
  CogIcon,
  CheckCircleIcon
} from '@heroicons/react/outline'
import Link from 'next/link'
import { useRouter } from 'next/router'

const navigation = [
  { name: 'Infrastructure', href: '/', icon: ViewGridIcon },
  { name: 'Users', href: '/users', icon: UsersIcon },
  { name: 'Permissions', href: '/permissions', icon: CheckCircleIcon },
  { name: 'Settings', href: '/settings', icon: CogIcon },
]

function classNames(...classes) {
  return classes.filter(Boolean).join(' ')
}

export default function Layout ({ children }) {
  const [sidebarOpen, setSidebarOpen] = useState(false)
  const router = useRouter()
  const [cookies] = useCookies(['token'])

  if (process.browser && !cookies.login) {
    router.replace("/login")
    return null
  }

  return (
    <div className="h-screen flex overflow-hidden bg-white">
      <Transition.Root show={sidebarOpen} as={Fragment}>
        <Dialog
          as="div"
          static
          className="fixed inset-0 flex z-40 md:hidden"
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
            <div className="relative flex-1 flex flex-col max-w-xs w-full bg-white">
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
              <div className="flex-1 h-0 pt-5 pb-4 overflow-y-auto">
                <div className="flex items-center flex-shrink-0 py-4 px-4 select-none">
                  <img src="/logo.svg" alt="infra"/>
                </div>
                <nav className="mt-5 px-2 space-y-1">
                  {navigation.map(item => (
                    <Link href={item.href} key={item.name}>
                      <a
                        className={classNames(
                          router.asPath === item.href
                            ? 'text-blue-600'
                            : 'text-gray-600 hover:text-gray-900',
                          'group flex items-center px-2 py-2 text-base font-semibold rounded-md'
                        )}
                      >
                        <item.icon
                          className={classNames(
                            router.asPath === item.href ? 'text-blue-600' : 'text-gray-400 group-hover:text-gray-500',
                            'mr-4 h-6 w-6'
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
          <div className="flex-shrink-0 w-14">{/* Force sidebar to shrink to fit close icon */}</div>
        </Dialog>
      </Transition.Root>
      <div className="hidden md:flex md:flex-shrink-0">
        <div className="flex flex-col w-64">
          <div className="flex flex-col h-0 flex-1 border-r border-gray-100 bg-white">
            <div className="flex-1 flex flex-col pt-5 pb-4 overflow-y-auto">
              <div className="flex items-center py-2 px-6 text-center select-none">
                <img src="/logo.svg" alt="infra" className="max-h-5"/>
              </div>
              <nav className="mt-5 flex-1 px-3 bg-white space-y-1 select-none">
                {navigation.map(item => (
                  <Link href={item.href} key={item.name}>
                    <a
                      className={classNames(
                        router.asPath === item.href ? 'text-blue-600' : 'text-gray-500 hover:text-gray-900',
                        'group flex items-center px-3 py-2 text-sm font-semibold rounded-md'
                      )}
                    >
                      <item.icon
                        className={classNames(
                          router.asPath === item.href ? 'text-blue-600' : 'text-gray-500 group-hover:text-gray-900',
                          'mr-2.5 h-5 w-5'
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
      <div className="flex flex-col w-0 flex-1 overflow-hidden bg-gray-50">
        <div className="md:hidden pl-1 pt-1 sm:pl-3 sm:pt-3">
          <button
            className="-ml-0.5 -mt-0.5 h-12 w-12 inline-flex items-center justify-center rounded-md text-gray-500 hover:text-gray-900 focus:outline-none focus:ring-2 focus:ring-inset focus:ring-blue-500"
            onClick={() => setSidebarOpen(true)}
          >
            <span className="sr-only">Open sidebar</span>
            <MenuIcon className="h-6 w-6" aria-hidden="true" />
          </button>
        </div>
        <main className="flex-1 relative z-0 overflow-y-auto focus:outline-none">
          {children}
        </main>
      </div>
    </div>
  )
}