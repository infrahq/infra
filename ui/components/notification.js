import { Fragment } from 'react'
import { Transition } from '@headlessui/react'
import { CheckCircleIcon } from '@heroicons/react/outline'
import { XIcon } from '@heroicons/react/solid'

export default function Notification({ show, text, setShow }) {
  return (
    <div
      aria-live='assertive'
      className='pointer-events-none fixed inset-0 z-10 flex items-end px-4 py-6 sm:items-end sm:p-6'
    >
      <div className='flex w-full flex-col items-center space-y-4 sm:items-end'>
        <Transition
          show={show}
          as={Fragment}
          enter='transform ease-out duration-300 transition'
          enterFrom='translate-y-2 scale-95 opacity-0 sm:translate-y-0 sm:translate-x-20'
          enterTo='translate-y-0 scale-100 opacity-100 sm:translate-x-0'
          leave='transition ease-in duration-100'
          leaveFrom='opacity-100 scale-100'
          leaveTo='opacity-0 scale-95'
        >
          <div className='pointer-events-auto w-full max-w-sm overflow-hidden rounded-lg bg-[#000] shadow-xl ring-1 ring-black ring-opacity-5'>
            <div className='p-4'>
              <div className='flex items-start'>
                <div className='flex-shrink-0'>
                  <CheckCircleIcon
                    className='h-6 w-6 text-green-400'
                    aria-hidden='true'
                  />
                </div>
                <div className='ml-3 w-0 flex-1 pt-0.5'>
                  <p className='text-sm font-medium text-white'>{text}</p>
                </div>
                <div className='ml-4 flex flex-shrink-0'>
                  <button
                    type='button'
                    className='inline-flex rounded-md text-zinc-600 hover:text-zinc-500'
                    onClick={() => setShow(false)}
                  >
                    <span className='sr-only'>Close</span>
                    <XIcon className='h-5 w-5' aria-hidden='true' />
                  </button>
                </div>
              </div>
            </div>
          </div>
        </Transition>
      </div>
    </div>
  )
}
