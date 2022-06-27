import { useEffect, useRef } from 'react'
import { XIcon } from '@heroicons/react/outline'
import ProfileIcon from './profile-icon'

export default function Sidebar({
  children,
  handleClose,
  title,
  iconPath,
  profileIcon,
}) {
  const ref = useRef()

  useEffect(() => ref.current.scrollTo(0, 0), [children])

  return (
    <aside
      ref={ref}
      className='my-0 flex h-full w-full max-w-sm flex-col overflow-x-visible overflow-y-scroll pl-8 pr-6 lg:max-w-md lg:pl-12 xl:max-w-lg xl:pl-16'
    >
      <header className='flex-start sticky top-0 z-10 flex items-center justify-between bg-black py-3'>
        <div className='mr-3 flex-none'>
          {profileIcon ? (
            <ProfileIcon name={profileIcon} />
          ) : (
            <div className='flex items-center rounded-md border border-violet-300/20 px-2 py-2 text-sm tracking-tight'>
              <img
                alt='profile icon'
                src={iconPath}
                className='h-5 w-5 opacity-50'
              />
            </div>
          )}
        </div>
        <h1 className='min-w-0 flex-1 flex-row items-center truncate text-2xs'>
          {title}
        </h1>
        <button
          type='button'
          className='bg-transparents -mr-2 flex-none cursor-pointer rounded-md p-2 text-gray-400 hover:text-white focus:outline-none'
          onClick={handleClose}
        >
          <XIcon className='h-6 w-6 stroke-1' aria-hidden='true' />
        </button>
      </header>
      {children}
    </aside>
  )
}
