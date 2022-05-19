import Link from 'next/link'
import { PlusIcon } from '@heroicons/react/outline'

export default function ({ title, subtitle, iconPath, buttonText, buttonHref }) {
  return (
    <div className='flex flex-col text-center py-32 mx-auto'>
      <img src={iconPath} className='mx-auto my-4 w-7 h-7' alt={title} />
      <h1 className='text-base font-bold mb-2'>{title}</h1>
      <h2 className='text-gray-300 mb-4 text-xs max-w-xs mx-auto'>{subtitle}</h2>
      {buttonHref && (
        <Link href={buttonHref}>
          <button className='flex items-center border border-violet-300 rounded-md px-4 py-2.5 my-2 mx-auto'>
            <PlusIcon className='w-2.5 h-2.5 mr-1' />
            <div className='text-violet-100 text-xs font-medium'>
              {buttonText}
            </div>
          </button>
        </Link>
      )}
    </div>
  )
}
