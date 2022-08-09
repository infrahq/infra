import { useEffect, useRef } from 'react'
import { XIcon } from '@heroicons/react/outline'

export default function Sidebar({
  children,
  onClose,
  title,
  iconPath,
  iconText,
}) {
  const ref = useRef()

  useEffect(() => ref.current.scrollTo(0, 0), [children])

  return (
    <aside
      ref={ref}
      className='my-0 flex h-full w-full min-w-[24em] max-w-sm flex-col overflow-y-auto overflow-x-visible pl-8 pr-6 lg:max-w-md lg:pl-12 xl:max-w-lg xl:pl-16'
    >
      <header className='flex-start sticky top-0 z-10 flex items-center justify-between bg-black py-3'>
        {iconPath && (
          <div className='flex h-7 w-7 flex-none items-center justify-center rounded-md border border-gray-800'>
            <img alt='PROFILE icon' className='h-3' src={iconPath} />
          </div>
        )}
        {iconText && (
          <div className='flex h-7 w-7 select-none items-center justify-center rounded-md border border-gray-800'>
            <span className='text-3xs font-normal leading-none'>
              {iconText}
            </span>
          </div>
        )}
        <div className='ml-3 min-w-0 flex-1 flex-row items-center truncate text-2xs'>
          {title}
        </div>
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
