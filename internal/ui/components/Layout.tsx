import { Fragment } from 'react'
import { Menu, Popover, Transition } from '@headlessui/react'
import { BellIcon, MenuIcon, XIcon } from '@heroicons/react/outline'
import { CogIcon, ViewGridIcon, UsersIcon } from '@heroicons/react/solid'
import classnames from 'classnames'
import Link from 'next/link'
import { useRouter } from 'next/router'
import { useCookies } from 'react-cookie'
import { V1 } from '../gen/v1.pb'

const user = {
  name: 'Chelsea Hagon',
  email: 'chelseahagon@example.com',
}

interface LayoutProps {
    children: JSX.Element[] | JSX.Element;
}

export default function Layout (props: LayoutProps): JSX.Element {
    const [cookies] = useCookies(['login'])
    const router = useRouter()

    if (process.browser && !cookies.login) {
        router.replace("/login")
        return <></>
    }

    const navigation = [
        { name: 'Infrastructure', href: '/', icon: ViewGridIcon },
        { name: 'Users', href: '/users', icon: UsersIcon },
        { name: 'Settings', href: '/settings', icon: CogIcon },
    ]
    
    const userNavigation = [{
        name: 'Logout',
        onClick: async () => {
            await V1.Logout({})
            router.replace("/login")
        }
    }]

    return (
        <div className="min-h-screen flex flex-col">
            {/* When the mobile menu is open, add `overflow-hidden` to the `body` element to prevent double scrollbars */}
            <Popover
                as="header"
                className={({ open }) =>
                    classnames(
                    open ? 'fixed inset-0 z-40 overflow-y-auto' : '',
                    'bg-white shadow lg:static lg:overflow-y-visible flex-none'
                    )
                }
            >
            {({ open }) => (
                <>
                <div className="px-4 py-3.5 sm:px-6">
                    <div className="relative flex justify-between">
                    <div className="flex lg:static xl:col-span-2">
                        <div className="flex-shrink-0 flex items-center">
                        <Link href="/">
                            <a>
                                <img
                                className="block fill-current text-gray-100"
                                style={{ height: "26px" }}
                                src="/combo.svg"
                                alt="Icon"
                                />
                            </a>
                        </Link>
                        </div>
                    </div>
                    <div className="flex items-center lg:hidden">
                        {/* Mobile menu button */}
                        <Popover.Button className="-mx-2 rounded-md p-2 inline-flex items-center justify-center text-gray-400 hover:bg-gray-100 hover:text-gray-500 focus:outline-none focus:ring-2 focus:ring-inset focus:ring-rose-500">
                        <span className="sr-only">Open menu</span>
                        {open ? (
                            <XIcon className="block h-6 w-6" aria-hidden="true" />
                        ) : (
                            <MenuIcon className="block h-6 w-6" aria-hidden="true" />
                        )}
                        </Popover.Button>
                    </div>
                    <div className="hidden lg:flex lg:items-center lg:justify-end xl:col-span-4">
                        <Menu as="div" className="flex-shrink-0 relative ml-5">
                        {({ open }) => (
                            <>
                            <div>
                                <Menu.Button className="bg-white rounded-full flex focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-600">
                                <span className="sr-only">Open user menu</span>
                                <span className="inline-block h-8 w-8 rounded-full overflow-hidden bg-gray-100">
                                    <svg className="h-full w-full text-gray-300" fill="currentColor" viewBox="0 0 24 24">
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
                                className="origin-top-right absolute z-10 right-0 mt-2 w-48 rounded-md cursor-pointer shadow-lg bg-black text-white ring-1 ring-black ring-opacity-5 py-1 focus:outline-none"
                                >
                                {userNavigation.map((item) => (
                                    <Menu.Item key={item.name}>
                                    {({ active }) => (
                                        <a
                                        onClick={item.onClick}
                                        className={classnames(
                                            active ? 'bg-gray-800' : '',
                                            'block py-2 px-4 text-white'
                                        )}
                                        >
                                        {item.name}
                                        </a>
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
                </div>

                <Popover.Panel as="nav" className="lg:hidden" aria-label="Global">
                    <div className="max-w-3xl mx-auto px-2 pt-2 pb-3 space-y-1 sm:px-4">
                    {navigation.map((item) => (
                        <Link href={item.href} key={item.name}>
                            <a
                            aria-current={router.asPath === item.href ? 'page' : undefined}
                            className={classnames(
                                router.asPath === item.href ? 'bg-gray-100 text-gray-900' : 'hover:bg-gray-50',
                                'block rounded-md py-2 px-3 text-base font-medium'
                            )}
                            >{item.name}</a>
                        </Link>
                    ))}
                    </div>
                    <div className="border-t border-gray-200 pt-4 pb-3">
                    <div className="max-w-3xl mx-auto px-4 flex items-center sm:px-6">
                        <div className="flex-shrink-0">
                        <span className="inline-block h-10 w-10 rounded-full overflow-hidden bg-gray-100">
                            <svg className="h-full w-full text-gray-300" fill="currentColor" viewBox="0 0 24 24">
                            <path d="M24 20.993V24H0v-2.996A14.977 14.977 0 0112.004 15c4.904 0 9.26 2.354 11.996 5.993zM16.002 8.999a4 4 0 11-8 0 4 4 0 018 0z" />
                            </svg>
                        </span>
                        </div>
                        <div className="ml-3">
                        <div className="text-base font-medium text-gray-800">{user.name}</div>
                        <div className="text-sm font-medium text-gray-500">{user.email}</div>
                        </div>
                        <button
                        type="button"
                        className="ml-auto flex-shrink-0 bg-white rounded-full p-1 text-gray-400 hover:text-gray-500 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-rose-500"
                        >
                        <span className="sr-only">View notifications</span>
                        <BellIcon className="h-6 w-6" aria-hidden="true" />
                        </button>
                    </div>
                    <div className="mt-3 max-w-3xl mx-auto px-2 space-y-1 sm:px-4">
                        {userNavigation.map((item) => (
                        <a
                            key={item.name}
                            onClick={item.onClick}
                            className="block rounded-md py-2 px-3 text-base font-medium text-gray-500 hover:bg-gray-50 hover:text-gray-900"
                        >
                            {item.name}
                        </a>
                        ))}
                    </div>
                    </div>
                </Popover.Panel>
                </>
            )}
            </Popover>

            <div className="flex flex-1 py-6">
            <div className="hidden lg:flex lg:flex-shrink-0">
                <div className="flex flex-col w-64">
                    <div className="flex flex-col h-0 flex-1 bg-white">
                        <div className="flex-1 flex flex-col pb-4 overflow-y-auto">
                            <nav className="flex-1 px-3 bg-white space-y-1 select-none">
                                {navigation.map(item => (
                                    <Link href={item.href} key={item.name}>
                                        <a
                                            className={classnames(
                                                router.asPath === item.href ? 'text-blue-600 bg-blue-50' : 'text-gray-600 hover:text-gray-800 hover:bg-gray-50',
                                                'group flex items-center px-3 py-2 font-medium rounded-lg'
                                            )}
                                        >
                                        <item.icon
                                            className={classnames(
                                            router.asPath === item.href ? 'text-blue-600' : 'text-gray-600 group-hover:text-gray-800',
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
            <div className="flex flex-col w-0 flex-1 overflow-hidden">
                <main className="flex-1 relative z-0 overflow-y-auto focus:outline-none">
                    {props.children}
                </main>
            </div>
            </div>
        </div>
    )
}
