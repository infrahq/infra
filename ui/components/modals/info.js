import { Dialog } from '@headlessui/react'

export default ({ header, children, handleCloseModal, modalOpen, iconPath }) => {
  return (
    <Dialog
      as='div'
      className='fixed z-10 inset-0 overflow-y-auto'
      open={modalOpen}
      onClose={handleCloseModal}
    >
      <div className='flex items-end justify-center min-h-screen pt-4 px-4 pb-20 text-center sm:block sm:p-0'>
        <Dialog.Overlay className='fixed inset-0 bg-black bg-opacity-75 transition-opacity' />
        <span className='hidden sm:inline-block sm:align-middle sm:h-screen' aria-hidden='true'>
          &#8203;
        </span>
        <div className='relative inline-block align-bottom bg-gradient-to-br from-violet-400/30 to-pink-200/30 rounded-xl text-left overflow-hidden shadow-xl transform transition-all sm:align-middle w-full max-w-3xl'>
          <div className='items-center justify-center bg-black rounded-xl m-0.5'>
            <div className='flex flex-row p-4'>
              <div className='hidden lg:flex self-start mr-4 bg-gradient-to-br from-violet-400/30 to-pink-200/30 rounded-full'>
                <div className='flex bg-black items-center justify-center rounded-full w-16 h-16 m-0.5'>
                  <img className='w-8 h-8' src={iconPath} />
                </div>
              </div>
              <div className='flex-1 flex flex-col space-y-4'>
                <Dialog.Title as='h1' className='text-2xl leading-6 font-bold text-white pt-6 pl-3'>
                  {header}
                </Dialog.Title>
                <div className='pb-3 pr-3 sm:pb-6 sm:pr-6'>
                  {children}
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </Dialog>
  )
}
