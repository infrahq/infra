import { Dialog } from '@headlessui/react'
import { ExclamationIcon } from '@heroicons/react/outline'

export default function ({ open, setOpen, onSubmit, title, message }) {
  return (
    <Dialog as='div' className='fixed z-10 inset-0 overflow-y-auto flex items-center justify-center min-h-screen pt-4 px-4 pb-20 text-center' open={open} onClose={() => setOpen && setOpen(false)}>
      <Dialog.Overlay className='fixed inset-0 bg-black bg-opacity-80 transition-opacity' />
      <aside className='relative inline-block rounded-lg text-left bg-black border border-gray-800 overflow-hidden transform transition-all align-middle max-w-xs w-full p-5'>
        <header className='flex items-center my-2 text-left'>
          <ExclamationIcon className='h-6 w-6 stroke-[1.5] text-pink-400 mr-2' aria-hidden='true' />
          <Dialog.Title as='h3' className='text-sm'>
            {title}
          </Dialog.Title>
        </header>
        <Dialog.Description className='text-2xs text-gray-300 my-12'>
          {message}
        </Dialog.Description>
        <footer className='mt-8 text-sm flex flex-row-reverse'>
          <button
            type='button'
            className='text-violet-200 border border-violet-200 focus:outline-violet-100 focus:text-white outline-offset-0 leading-none rounded-md text-2xs ml-3 px-8'
            onClick={() => onSubmit()}
          >
            Delete
          </button>
          <button
            type='button'
            className='px-8 py-2 text-3xs text-gray-400 uppercase focus:outline-none focus:text-gray-100'
            onClick={() => setOpen && setOpen(false)}
          >
            Cancel
          </button>
        </footer>
      </aside>
    </Dialog>
  )
}
