import { Dialog } from '@headlessui/react'

export default ({ header, children, handleCloseModal, modalOpen }) => {
  return (
    <Dialog
      as='div'
      className='fixed z-10 inset-0 overflow-y-auto'
      open={modalOpen}
      onClose={handleCloseModal}
    >
      <div className='flex items-end justify-center min-h-screen pt-4 px-4 pb-20 text-center sm:block sm:p-0'>
        <Dialog.Overlay className='fixed inset-0 bg-gray-500 bg-opacity-25 transition-opacity' />
        <span className='hidden sm:inline-block sm:align-middle sm:h-screen' aria-hidden='true'>
          &#8203;
        </span>
        <div className='relative inline-block align-bottom bg-gradient-to-br from-violet-400/30 to-pink-200/30 rounded-xl text-left overflow-hidden shadow-xl transform transition-all sm:align-middle sm:w-3/4'>
          <div className='items-center justify-center bg-black rounded-xl m-0.5'>
            <Dialog.Title as='div' className='pt-2 px-2 sm:pt-6 sm:px-6'>
              <div className='flex fles-start items-center'>
                <div className='lg:flex self-start mt-4 mr-8 bg-gradient-to-br from-violet-400/30 to-pink-200/30 rounded-full'>
                  <div className='flex bg-black items-center justify-center rounded-full w-16 h-16 m-0.5'>
                    <img className='w-8 h-8' src='/grant-access-color.svg' />
                  </div>
                </div>
                <h3 className='text-lg leading-6 font-bold text-white'>{header}</h3>
              </div>
            </Dialog.Title>
            <div className='pt-1.5 pb-6 pl-12 pr-6 sm:pt-3 sm:pb-12 sm:pl-24 sm:pr-12'>
              {children}
            </div>
          </div>
        </div>
      </div>
    </Dialog>
  )
}
