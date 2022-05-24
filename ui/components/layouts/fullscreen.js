import Link from 'next/link'
import { XIcon, ChevronLeftIcon } from '@heroicons/react/outline'

export default function ({ children, backHref, closeHref }) {
  return (
    <div className='flex flex-col w-full h-full'>
      <div className={`flex flex-none text-right justify-end ${backHref ? 'justify-between' : 'justify-end'}`}>
        {backHref && (
          <Link href={backHref || '/'}>
            <a className='flex items-center p-4 text-gray-400 hover:text-white text-3xs uppercase'>
              <ChevronLeftIcon className='w-5 h-5 mr-1.5 stroke-1' />Back
            </a>
          </Link>
        )}
        <Link href={closeHref || '/'}>
          <a className='flex items-center p-4 text-gray-400 hover:text-white text-3xs uppercase'>
            Close<XIcon className='w-6 h-6 ml-1 stroke-1' />
          </a>
        </Link>
      </div>
      <div className='flex-1 flex justify-center items-center mb-10'>
        <div className='w-full max-w-xs border rounded-lg border-gray-800'>
          {children}
        </div>
      </div>
    </div>
  )
}
