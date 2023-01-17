import { Fragment, useEffect, useState } from 'react'
import Link from 'next/link'
import { useRouter } from 'next/router'
import { Dialog, Transition } from '@headlessui/react'
import {
  CpuChipIcon,
  UserGroupIcon,
  UserIcon,
  Cog8ToothIcon,
  XMarkIcon,
  Bars3Icon,
  UserCircleIcon,
  ArrowLeftOnRectangleIcon,
} from '@heroicons/react/24/outline'

import { useUser } from '../../lib/hooks'

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
                    <XMarkIcon
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

export default function Dashboard({ children }) {
  const router = useRouter()

  const { user, loading, isAdmin, isAdminLoading, org, logout } = useUser()
  const [sidebarOpen, setSidebarOpen] = useState(false)

  useEffect(() => {
    if (loading) {
      return
    }

    if (!user) {
      router.replace(
        router.asPath === '/'
          ? '/login'
          : `/login?next=${encodeURIComponent(router.asPath)}`
      )
    }
  }, [loading])

  if (loading || !user) {
    return null
  }

  if (isAdminLoading) {
    return null
  }

  const navigation = [
    {
      name: 'Infrastructure',
      href: '/destinations',
      icon: CpuChipIcon,
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
      icon: Cog8ToothIcon,
    },
  ]

  for (const n of [...navigation]) {
    if (router.pathname.startsWith(n.href) && n.admin && !isAdmin) {
      router.replace('/')
      return null
    }
  }

  function Nav() {
    return (
      <>
        <div className='mb-2 flex flex-shrink-0 select-none items-center px-3'>
          <Link href='/destinations'>
            <img className='my-2 h-7' src='/logo.svg' alt='Infra' />
          </Link>
        </div>
        <div className='mt-5 h-0 flex-1 overflow-y-auto'>
          <nav className='flex-1 space-y-1'>
            {navigation
              ?.filter(n => (n.admin ? isAdmin : true))
              .map(item => (
                <Link
                  key={item.name}
                  href={item.href}
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
                </Link>
              ))}
          </nav>
        </div>
        <div className='space-y-1 px-2'>
          <div className='flex space-x-3'>
            <UserCircleIcon className='h-[18px] w-[18px] flex-none text-blue-500' />
            <div className='min-w-0 flex-1 '>
              <p
                className='truncate text-xs font-semibold leading-none text-gray-900'
                title={user?.name}
              >
                {user?.name}
              </p>
              <p
                className='truncate text-[12px] text-gray-600'
                title={org?.name}
              >
                {org?.name}
              </p>
            </div>
          </div>
          <button
            type='button'
            onClick={async () => {
              await logout()
              router.replace('/login')
            }}
            className='flex w-full cursor-pointer items-center text-[12px] font-medium text-gray-500/75 hover:text-gray-500'
          >
            <ArrowLeftOnRectangleIcon
              className='mr-3 h-[18px] w-[18px]'
              aria-hidden='true'
            />
            Log out
          </button>
        </div>
      </>
    )
  }

  return (
    <div className='relative flex'>
      <>
        <SidebarNav open={sidebarOpen} setOpen={setSidebarOpen}>
          <Nav navigation={navigation} admin={isAdmin} />
        </SidebarNav>
        <div className='sticky top-0 hidden h-screen w-48 flex-none flex-col border-r border-gray-100 px-3 pt-5 pb-4 md:flex lg:w-60'>
          <Nav navigation={navigation} admin={isAdmin} />
        </div>
      </>

      {/* Main content */}
      <div className='mx-auto flex min-w-0 flex-1 flex-col'>
        <div className='sticky top-0 z-20 flex flex-shrink-0 border-b border-gray-100 bg-white/90 px-6 pl-2 backdrop-blur-lg md:hidden md:py-2 md:px-6'>
          <button
            type='button'
            className='p-4 text-black focus:outline-none focus:ring-2 focus:ring-inset focus:ring-blue-500 md:hidden'
            onClick={() => setSidebarOpen(true)}
          >
            <span className='sr-only'>Open sidebar</span>
            <Bars3Icon className='h-6 w-6' aria-hidden='true' />
          </button>
        </div>

        <main className='mx-auto w-full max-w-6xl flex-1 px-6 py-8'>
          {children}
        </main>
      </div>
    </div>
  )
}
