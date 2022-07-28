import { Fragment } from 'react'
import { Transition } from '@headlessui/react'
import { CheckCircleIcon } from '@heroicons/react/outline'
import { XIcon } from '@heroicons/react/solid'

export default function Notification({ show, setShow, text }) {
  return (
    <div
      aria-live='assertive'
      className='pointer-events-none fixed inset-0 flex items-end px-4 py-6 sm:items-start sm:p-6'
    >
      <div className='fixed bottom-6 right-6 flex w-full flex-col items-center space-y-4 sm:items-end'>
        <Transition
          show={show}
          as={Fragment}
          enter='transform ease-out duration-300 transition'
          enterFrom='translate-y-2 opacity-0 sm:translate-y-0 sm:translate-x-2'
          enterTo='translate-y-0 opacity-100 sm:translate-x-0'
          leave='transition ease-in duration-100'
          leaveFrom='opacity-100'
          leaveTo='opacity-0'
        >
          <div className='pointer-events-auto w-full max-w-sm overflow-hidden rounded-lg bg-gray-900 shadow-lg ring-1 ring-black ring-opacity-5'>
            <div className='p-4'>
              <div className='flex items-start'>
                <div className='flex-shrink-0'>
                  <CheckCircleIcon
                    className='h-6 w-6 text-white'
                    aria-hidden='true'
                  />
                </div>
                <div className='ml-3 w-0 flex-1 pt-0.5'>
                  <p className='text-sm text-white'>{text}</p>
                </div>
                <div className='ml-4 flex flex-shrink-0'>
                  <button
                    data-testid='notification-remove-button'
                    type='button'
                    className='inline-flex rounded-md text-gray-400 hover:text-gray-500 focus:outline-none'
                    onClick={() => setShow(false)}
                  >
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
