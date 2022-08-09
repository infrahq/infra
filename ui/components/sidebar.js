import { useEffect, useRef } from 'react'
import { XIcon } from '@heroicons/react/outline'
import ProfileIcon from './profile-icon'

export default function Sidebar({
  children,
  onClose,
  title,
  iconPath,
  iconText,
  // profileIcon,
}) {
  const ref = useRef()

  useEffect(() => ref.current.scrollTo(0, 0), [children])

  return (
    <aside
      ref={ref}
      className='my-0 flex h-full w-full min-w-[24em] max-w-sm flex-col overflow-y-auto overflow-x-visible pl-8 pr-6 lg:max-w-md lg:pl-12 xl:max-w-lg xl:pl-16'
    >
      <header className='flex-start sticky top-0 z-10 flex items-center justify-between bg-black py-3'>
        {(iconPath || iconText) && (
          <div className='mr-3 flex items-center'>
            {/* {profileIcon ? (
            <ProfileIcon name={profileIcon} />
          ) : ( */}

            <div className='flex h-7 w-7 flex-none items-center rounded-md border border-gray-800 px-2 py-2 text-sm tracking-tight'>
              {iconPath && (
                <img
                  alt='profile icon'
                  src={iconPath}
                  className='h-5 opacity-50'
                />
              )}
              {iconText && (
                <span className='text-3xs font-normal leading-none'>
                  {iconText}
                </span>
              )}
            </div>

            {/* )} */}
          </div>
        )}
        <h1 className='min-w-0 flex-1 flex-row items-center truncate text-2xs'>
          {title}
        </h1>
        <button
          type='button'
          className='bg-transparents -mr-2 flex-none cursor-pointer rounded-md p-2 text-gray-400 hover:text-white focus:outline-none'
          onClick={onClose}
        >
          <XIcon className='h-6 w-6 stroke-1' aria-hidden='true' />
        </button>
      </header>
      {children}
    </aside>
  )
}
