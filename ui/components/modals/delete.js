import { Fragment, useRef } from 'react'
import { Dialog, Transition } from '@headlessui/react'
import { ExclamationIcon } from '@heroicons/react/outline'

export default function ({ open, setOpen, onSubmit, title, message }) {
  const cancelButtonRef = useRef(null)

  return (
    <Transition.Root show={open} as={Fragment}>
      <Dialog as='div' className='fixed z-10 inset-0 overflow-y-auto' initialFocus={cancelButtonRef} onClose={setOpen}>
        <div className='flex items-end justify-center min-h-screen pt-4 px-4 pb-20 text-center sm:block sm:p-0'>
          <Dialog.Overlay className='fixed inset-0 bg-black bg-opacity-75 transition-opacity' />

          {/* This element is to trick the browser into centering the modal contents. */}
          <span className='hidden sm:inline-block sm:align-middle sm:h-screen' aria-hidden='true'>
            &#8203;
          </span>
          <div className='relative inline-block bg-gradient-to-tr from-[#B06363] to-[#FF00C7] rounded-3xl text-left overflow-hidden shadow-xl transform transition-all my-8 align-middle max-w-2xl w-full p-px'>
            <div className='bg-black px-10 pt-12 pb-8 rounded-3xl'>
              <div className='flex items-start'>
                <div className='rounded-full bg-gradient-to-tr from-[#B06363] to-[#FF00C7]'>
                  <div className='flex h-14 w-14 items-center justify-center rounded-full bg-black m-0.5'>
                    <ExclamationIcon className='h-6 w-6 text-[#D3398F]' aria-hidden='true' />
                  </div>
                </div>
                <div className='mt-1 ml-5 text-left'>
                  <Dialog.Title as='h3' className='text-base leading-6 font-bold text-white'>
                    {title}
                  </Dialog.Title>
                  <p className='text-sm text-gray-400 my-0.5'>
                    {message}
                  </p>
                </div>
              </div>

              {/* buttons */}
              <div className='mt-8 text-sm flex flex-row-reverse'>
                <button
                  type='button'
                  className='w-auto inline-flex justify-center rounded-full bg-gradient-to-tr from-[#B06363] to-[#FF00C7] font-medium text-white focus:outline-none focus:ring-2 focus:ring-[#FF00C7] ml-3'
                  onClick={() => onSubmit()}
                >
                  <div className='bg-black  px-10 py-3.5 rounded-full m-0.5'>
                    Delete
                  </div>
                </button>
                <button
                  type='button'
                  className='w-auto inline-flex items-center justify-center rounded-full px-10 py-3.5 bg-black hover:opacity-75 font-medium text-white focus:outline-none focus:ring-2 focus:ring-zinc-600'
                  onClick={() => setOpen(false)}
                  ref={cancelButtonRef}
                >
                  Cancel
                </button>
              </div>
            </div>
          </div>
        </div>
      </Dialog>
    </Transition.Root>
  )
}
