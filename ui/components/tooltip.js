export default function Tooltip({ children, message }) {
  return (
    <div className='group relative flex flex-col items-center'>
      {children}
      <div className='absolute left-0 bottom-0 mb-6 hidden flex-col items-center group-hover:flex'>
        <span className='whitespace-no-wrap relative z-10 w-[20rem] rounded-md bg-black p-2 text-xs leading-none text-white shadow-lg'>
          {message}
        </span>
      </div>
    </div>
  )
}
