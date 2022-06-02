import { useEffect, useRef } from 'react'
import { XIcon } from '@heroicons/react/outline'
import ProfileIcon from './profile-icon'

export default ({ children, handleClose, title, iconPath, profileIcon }) => {
  const ref = useRef()

  useEffect(() => ref.current.scrollTo(0, 0), [children])

  return (
    <aside ref={ref} className='flex w-full h-full max-w-xs lg:max-w-md ml-20 pr-6 my-0 flex-col overflow-y-scroll'>
      <header className='flex flex-start justify-between items-center z-10 py-3 sticky top-0 bg-black'>
        <div className='flex items-center space-x-3'>
          {profileIcon
            ? <ProfileIcon name={profileIcon} />
            : (
              <div className='border border-violet-300/20 rounded-md flex items-center tracking-tight text-sm px-2 py-2'>
                <img src={iconPath} className='w-5 h-5 opacity-50' />
              </div>
              )}
          <h1 className='flex flex-row items-center space-x-4 text-2xs'>{title}</h1>
        </div>
        <button
          type='button'
          className='rounded-md bg-transparents text-gray-400 p-2 -mr-2 hover:text-white focus:outline-none cursor-pointer'
          onClick={handleClose}
        >
          <XIcon className='h-6 w-6 stroke-1' aria-hidden='true' />
        </button>
      </header>
      {children}
    </aside>
  )
}
