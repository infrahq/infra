export default function ProfileIcon({ name }) {
  return (
    <div className='flex h-7 w-7 flex-none items-stretch self-start rounded-md border border-violet-300/40'>
      <div className='relative m-0.5 flex flex-1 select-none items-center justify-center rounded-[4px] border border-violet-300/70 text-center text-3xs font-normal leading-none'>
        <span className='absolute inset-x-0 -mt-[1px]'>{name}</span>
      </div>
    </div>
  )
}
