export default function ({ children }) {
  return (
    <div className='w-full min-h-full flex flex-col justify-center'>
      <div className='flex flex-col w-full max-w-xs mx-auto justify-center items-center px-5 pt-8 pb-4 border rounded-lg border-gray-800'>
        <div className='border border-violet-200/25 rounded-full p-2.5 mb-4'>
          <img className='w-12 h-12' src='/infra-color.svg' />
        </div>
        {children}
      </div>
    </div>
  )
}
