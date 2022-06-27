export default function HeaderIcon({ iconPath, size = 8, position }) {
  return (
    <div
      className={`flex items-center justify-center rounded-full bg-gradient-to-br from-violet-400/30 to-pink-200/30 ${
        position === 'center' ? 'mx-auto my-4' : 'mt-6 mb-4'
      }`}
    >
      <div className='m-0.5 flex h-16 w-16 items-center justify-center rounded-full bg-black'>
        <img
          alt='header icon'
          className={`w-${size} h-${size}`}
          src={iconPath}
        />
      </div>
    </div>
  )
}
