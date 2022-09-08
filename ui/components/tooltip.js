export default function Tooltip({ children, message, direction = 'right' }) {
  return (
    <div className='group relative flex'>
      {children}
      <div className='absolute bottom-0 mb-6 hidden flex-col group-hover:flex'>
        <span
          className={`absolute ${
            direction === 'left' ? '-left-60' : '-left-0'
          } -top-2 z-10 hidden w-[20rem] -translate-y-full rounded-lg bg-black p-2 text-left text-xs text-white group-hover:flex`}
        >
          {message}
        </span>
      </div>
    </div>
  )
}
