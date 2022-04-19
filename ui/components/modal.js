export default ({header, children, handleCloseModal }) => {
  return (
    <>
    <div className="justify-center items-center flex overflow-x-hidden overflow-y-auto fixed inset-0 z-50 outline-none focus:outline-none">
      <div className="w-2/4">
        <div className="border-0 rounded-lg shadow-lg relative flex flex-col bg-black outline-none focus:outline-none">
          <div className="px-3 flex items-center justify-between">
            <h3 className="text-lg font-normal">{header}</h3>
            <button
              className="p-1 ml-auto bg-transparent border-0 text-white float-right text-lg leading-none font-semibold outline-none focus:outline-none"
              onClick={handleCloseModal}
            >
              &#10005;
            </button>
          </div>
          {/*body*/}
          <div className="relative p-3 flex-auto">
            <div className="my-4 text-slate-500 text-lg leading-relaxed">
              {children}
            </div>
          </div>
        </div>
      </div>
    </div>
    <div className="opacity-75 fixed inset-0 z-40 bg-black"></div>
  </>
  )
}