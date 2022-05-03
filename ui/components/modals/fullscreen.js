import Link from 'next/link'
import { XIcon, ArrowLeftIcon } from '@heroicons/react/outline'

export default function ({ children, closeHref, backHref, verticalCenteredContent = true }) {
  return (
    <div className='flex flex-col w-full h-full'>
      <div className={`flex flex-none text-right ${backHref ? 'justify-between' : 'justify-end'}`}>
        {backHref && (
          <Link href={backHref || '/'}>
            <a className='flex items-center px-4 text-gray-300 hover:text-white'>
              <ArrowLeftIcon className='w-4 h-4' /><div className='text-sm ml-2'>Back</div>
            </a>
          </Link>
        )}
        <Link href={closeHref || '/'}>
          <a className='flex items-center p-4 text-gray-300 hover:text-white'>
            <div className='text-sm mr-2'>Close</div><XIcon className='w-5 h-5' />
          </a>
        </Link>
      </div>
      <div className={`flex-1 flex justify-center ${verticalCenteredContent ? 'items-center' : ''}`}>
        {children}
      </div>
    </div>
  )
}
