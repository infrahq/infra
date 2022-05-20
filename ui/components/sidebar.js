import { XIcon } from '@heroicons/react/outline'

export default ({ children, handleClose, title, iconPath }) => {
  return (
    <aside className='flex w-full h-full max-w-md ml-20 my-0 flex-col'>
      <header className='flex flex-start justify-between items-center my-3'>
        <div className='flex items-center'>
          <div className='border border-violet-300/20 rounded-md flex items-center tracking-tight text-sm mr-2 px-2 py-2'>
            <img src={iconPath} className='w-5 h-5 opacity-50' />
          </div>
          <h1 className='flex flex-row items-center space-x-4 text-xs'>{title}</h1>
        </div>
        <button
            type='button'
            className='rounded-md bg-transparents text-gray-400 hover:text-white focus:outline-none cursor-pointer'
            onClick={handleClose}
          >
          <XIcon className='h-5 w-5' aria-hidden='true' />
        </button>
      </header>
      {children}
    </aside>
  )
}
