export default function ({ iconPath }) {
  return (
    <div className='hidden lg:flex self-start mt-2 mr-8 bg-gradient-to-br from-violet-400/30 to-pink-200/30 items-center justify-center rounded-full'>
      <div className='flex bg-black items-center justify-center rounded-full w-16 h-16 m-0.5'>
        <img className='w-8 h-8' src={iconPath} />
      </div>
    </div>
  )
}