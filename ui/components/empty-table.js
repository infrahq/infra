import Link from 'next/link'
import { PlusIcon } from '@heroicons/react/outline'

export default function ({ title, subtitle, iconPath, buttonText, buttonHref }) {
  return (
    <div className='flex flex-col text-center py-32 mx-auto'>
      <img src={iconPath} className='mx-auto my-4 w-7 h-7' alt={title} />
      <h1 className='text-base font-bold mb-2'>{title}</h1>
      <h2 className='text-gray-300 mb-4 text-name max-w-xs mx-auto'>{subtitle}</h2>
      {buttonHref && (
        <Link href={buttonHref}>
          <button className='bg-gradient-to-tr from-indigo-300 to-pink-100 hover:from-indigo-200 hover:to-pink-50 rounded-md p-0.5 my-2 mx-auto'>
            <div className='bg-black rounded-md flex items-center tracking-tight text-sm px-6 py-3'>
              <PlusIcon className='w-2 h-2 mr-1' />
              <div className='text-purple-50'>
                {buttonText}
              </div>
            </div>
          </button>
        </Link>
      )}
    </div>
  )
}
