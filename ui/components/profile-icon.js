export default function ({ name }) {
  return (
    <div className='flex flex-none self-start items-stretch border border-violet-300/40 rounded-md w-7 h-7'>
      <div className='relative text-center flex flex-1 justify-center items-center border border-violet-300/70 text-3xs rounded-[4px] leading-none font-normal m-0.5 select-none'>
        <span className='absolute inset-x-0 -mt-[1px]'>{name}</span>
      </div>
    </div>
  )
}
