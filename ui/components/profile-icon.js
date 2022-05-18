export default function ({ name, hasOpacity = true }) {
  return (
    <div className='bg-gradient-to-tr from-indigo-300/20 to-pink-100/20 rounded-lg p-px'>
      <div className='bg-black flex-none flex items-center justify-center w-8 h-8 rounded-lg'>
        <div className={`bg-gradient-to-tr ${hasOpacity ? 'from-indigo-300/40 to-pink-100/40 ' : 'from-indigo-300 to-pink-100'} rounded-[4px] p-px`}>
          <div className={`bg-black flex-none text-subtitle ${hasOpacity ? 'text-gray-500' : 'text-white'} flex justify-center items-center w-6 h-6 font-bold rounded-[4px]`}>
            {name}
          </div>
        </div>
      </div>
    </div>
  )
}
