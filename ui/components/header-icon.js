export default function ({ iconPath, size = 8, position }) {
  return (
    <div className={`flex items-center justify-center bg-gradient-to-br from-violet-400/30 to-pink-200/30 rounded-full ${position === 'center' ? 'mx-auto my-4' : 'mt-6 mb-4'}`}>
      <div className='flex bg-black items-center justify-center rounded-full w-16 h-16 m-0.5'>
        <img className={`w-${size} h-${size}`} src={iconPath} />
      </div>
    </div>
  )
}
