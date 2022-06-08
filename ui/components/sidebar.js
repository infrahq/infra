import { useEffect, useRef } from 'react'
import { Menu } from '@headlessui/react'
import { XIcon, DotsHorizontalIcon } from '@heroicons/react/outline'
import ProfileIcon from './profile-icon'

function ActionBtn ({ remove, edit }) {
  return (
    <Menu as="div" className="relative text-left">
      <div>
        <Menu.Button className='rounded-md bg-transparents text-gray-400 p-2 -mr-2 hover:text-white focus:outline-none cursor-pointer'>
          <span className="sr-only">Open options</span>
          <DotsHorizontalIcon className='h-6 w-6 stroke-1' aria-hidden='true' />
        </Menu.Button>
      </div>

        <Menu.Items className="origin-top-right absolute right-0 mt-2 w-52 rounded-md shadow-lg bg-gray-900 border border-gray-800 focus:outline-none">
          <div className="py-1">
            <Menu.Item>
              {({ active }) => (
                <a
                  onClick={edit}
                  className={`block px-4 py-2 text-4xs uppercase cursor-pointer ${active ? 'text-gray-400' : 'text-white'}`}
                >
                  Edit
                </a>
              )}
            </Menu.Item>
            <Menu.Item>
              {({ active }) => (
                <a
                  onClick={remove}
                  className={` block px-4 py-2 text-4xs uppercase cursor-pointer ${active ? 'text-gray-400' : 'text-white'}`}
                >
                  Remove
                </a>
              )}
            </Menu.Item>
          </div>
        </Menu.Items>
    </Menu>
  )
}

export default ({ children, handleClose, title, iconPath, profileIcon, showActionBtn = false, remove, edit}) => {
  const ref = useRef()

  useEffect(() => ref.current.scrollTo(0, 0), [children])

  return (
    <aside ref={ref} className='flex w-full h-full max-w-xs lg:max-w-sm xl:max-w-md ml-20 pr-6 my-0 flex-col overflow-y-scroll'>
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
        <div className='flex space-x-1'>
          {/* {showActionBtn && <button
            type='button'
            className='rounded-md bg-transparents text-gray-400 p-2 -mr-2 hover:text-white focus:outline-none cursor-pointer'
            onClick={handleClose}
          >
            <DotsHorizontalIcon className='h-6 w-6 stroke-1' aria-hidden='true' />
          </button>} */}
          {showActionBtn && <ActionBtn remove={remove} edit={edit} />}
          <button
            type='button'
            className='rounded-md bg-transparents text-gray-400 p-2 -mr-2 hover:text-white focus:outline-none cursor-pointer'
            onClick={handleClose}
          >
            <XIcon className='h-6 w-6 stroke-1' aria-hidden='true' />
          </button>
        </div>
      </header>
      {children}
    </aside>
  )
}
