import { Dialog } from '@headlessui/react'
import { ExclamationIcon } from '@heroicons/react/outline'

export default function DeleteModal({
  open,
  setOpen,
  onSubmit,
  title,
  message,
  primaryButtonText = 'Remove',
}) {
  return (
    <Dialog
      as='div'
      className='fixed inset-0 z-10 flex min-h-screen items-center justify-center overflow-y-auto px-4 pt-4 pb-20 text-center'
      open={open}
      onClose={() => setOpen && setOpen(false)}
    >
      <Dialog.Overlay className='fixed inset-0 bg-black bg-opacity-80 transition-opacity' />
      <aside className='relative inline-block w-full max-w-md transform overflow-hidden rounded-lg border border-gray-800 bg-black p-5 text-left align-middle transition-all'>
        <header className='my-2 flex items-center text-left'>
          <ExclamationIcon
            className='mr-2 h-6 w-6 stroke-[1.5] text-pink-400'
            aria-hidden='true'
          />
          <Dialog.Title as='h3' className='text-2xs'>
            {title}
          </Dialog.Title>
        </header>
        <Dialog.Description className='my-7 ml-8 text-2xs text-gray-400'>
          {message}
        </Dialog.Description>
        <footer className='mt-8 flex flex-row-reverse text-sm'>
          <button
            type='button'
            className='rounded-md border border-violet-300 px-8 text-2xs leading-none text-violet-100 outline-offset-0 focus:text-white focus:outline-violet-100'
            onClick={() => onSubmit()}
          >
            {primaryButtonText}
          </button>
          <button
            type='button'
            className='px-8 py-2 text-4xs uppercase text-gray-400 focus:text-gray-100 focus:outline-none'
            onClick={() => setOpen && setOpen(false)}
          >
            Cancel
          </button>
        </footer>
      </aside>
    </Dialog>
  )
}
