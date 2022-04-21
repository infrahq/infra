export default function ({ iconPath, width=8, position }) {
  return (
    <div className={`flex items-center justify-center bg-gradient-to-br from-violet-400/30 to-pink-200/30 rounded-full ${position === 'center' ? 'mx-auto my-4' : ''}`}>
      <div className='flex bg-black items-center justify-center rounded-full w-16 h-16 m-0.5'>
        <img className={`w-${width} h-${width}`} src={iconPath} />
      </div>
    </div>
  )
}