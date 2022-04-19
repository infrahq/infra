import { Dialog } from '@headlessui/react'

export default ({ header, children, handleCloseModal, modalOpen }) => {
  return (
  //   <>
  //   <div className="justify-center items-center flex overflow-x-hidden overflow-y-auto fixed inset-0 z-50 outline-none focus:outline-none">
  //     <div className="w-2/4">
  //       <div className="border-0 rounded-lg shadow-lg relative flex flex-col bg-black outline-none focus:outline-none">
  //         <div className="px-3 flex items-center justify-between">
  //           <h3 className="text-lg font-normal">{header}</h3>
  //           <button
  //             className="p-1 ml-auto bg-transparent border-0 text-white float-right text-lg leading-none font-semibold outline-none focus:outline-none"
  //             onClick={handleCloseModal}
  //           >
  //             &#10005;
  //           </button>
  //         </div>
  //         {/*body*/}
  //         <div className="relative p-3 flex-auto">
  //           <div className="my-4 text-slate-500 text-lg leading-relaxed">
  //             {children}
  //           </div>
  //         </div>
  //       </div>
  //     </div>
  //   </div>
  //   <div className="opacity-75 fixed inset-0 z-40 bg-black"></div>
  // </>
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
        <div className='relative inline-block align-bottom bg-black rounded-lg px-4 pt-5 pb-4 text-left overflow-hidden shadow-xl transform transition-all sm:my-8 sm:align-middle sm:max-w-lg sm:w-full sm:p-6'>
          <Dialog.Title>{header}</Dialog.Title>
          <div className='mt-2'>
            {children}
          </div>
        </div>
      </div>
    </Dialog>
  )
}
