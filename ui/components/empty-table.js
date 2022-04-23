import Link from 'next/link'

import HeaderIcon from './header-icon'

export default function ({ title, subtitle, iconPath, buttonText, buttonHref }) {
  return (
    <div className='flex flex-col text-center my-24'>
      <HeaderIcon iconPath={iconPath} position='center' />
      <h1 className='text-white text-lg font-bold mb-2'>{title}</h1>
      <h2 className='text-gray-300 mb-4 text-sm max-w-xs mx-auto'>{subtitle}</h2>
      <Link href={buttonHref}>
        <button className='bg-gradient-to-tr from-indigo-300 to-pink-100 hover:from-indigo-200 hover:to-pink-50 rounded-full p-0.5 my-2 mx-auto'>
          <div className='bg-black rounded-full flex items-center tracking-tight text-sm px-6 py-3'>
            {buttonText}
          </div>
        </button>
      </Link>
    </div>
  )
}
