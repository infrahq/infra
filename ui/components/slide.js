import { Fragment } from 'react'
import { Dialog, Transition } from '@headlessui/react'
import { XIcon } from '@heroicons/react/outline'

export default ({ children, open, handleClose, title, iconPath, footerBtns, deleteModalShown }) => {

  return (
    <Dialog as="div" className={`relative ${deleteModalShown ? '' : 'z-10'}`} onClose={handleClose} open={open}>
      <div className="fixed inset-0 overflow-hidden">
        <div className="absolute inset-0 overflow-hidden">
          <div className="pointer-events-none fixed inset-y-0 right-0 flex max-w-full pl-10">
              <Dialog.Panel className="pointer-events-auto w-screen max-w-lg">
                <div className="flex h-full flex-col bg-black shadow-xl">
                  <div className="flex min-h-0 flex-1 flex-col overflow-y-scroll py-4">
                    <div className="px-4 sm:px-6">
                      <div className="flex flex-start justify-between">
                      <Dialog.Title className="flex flex-row items-center space-x-4">
                          <div className='bg-gradient-to-tr from-indigo-300/20 to-pink-100/20 rounded-md p-0.5 my-2 mx-auto'>
                            <div className='bg-black rounded-md flex items-center tracking-tight text-sm px-2 py-2'>
                              <img src={iconPath} className='w-4 h-4' />
                            </div>
                          </div>
                          <div className='text-subtitle text-white'>{title}</div>
                        </Dialog.Title>
                        <div className="ml-3 flex h-7 items-center">
                        <button
                            type="button"
                            className="rounded-md bg-transparents text-gray-400 hover:text-white focus:outline-none cursor-pointer"
                            onClick={handleClose}
                          >
                            <XIcon className="h-5 w-5" aria-hidden="true" />
                          </button>
                        </div>
                      </div>
                    </div>
                    <div className="relative mt-6 flex-1 px-4 sm:px-6">
                      {children}
                    </div>
                  </div>
                  <div className="flex flex-shrink-0 justify-end px-4 py-4">
                    {footerBtns && <div className="flex flex-shrink-0 justify-end px-4 py-4">
                      {footerBtns.map((b => (
                        <button key={b.text} type='button' onClick={b.handleOnClick} className='bg-gradient-to-tr cursor-pointer from-indigo-300 to-pink-100 hover:from-indigo-200 hover:to-pink-50 rounded-md p-0.5'>
                          <div className='bg-black rounded-md flex items-center text-name px-6 py-3'>
                            <div className='text-purple-50'>
                              {b.text}
                            </div>
                          </div>
                        </button>
                      )))}
                    </div>}
                  </div>
                </div>
              </Dialog.Panel>
          </div>
        </div>
      </div>
    </Dialog>
  )
}