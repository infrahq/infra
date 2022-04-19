import Link from 'next/link'
import { XIcon, ArrowLeftIcon } from '@heroicons/react/outline'

export default function ({ children, closeHref, backHref }) {
  return (
    <div className='flex flex-col w-full h-full'>
      <div className={`flex flex-none text-right ${backHref ? 'justify-between' : 'justify-end'}`}>
        {backHref && (
          <Link href={backHref || '/'}>
            <a className='flex items-center text-gray-400 px-4'>
              <ArrowLeftIcon className='w-3 h-3 mr-1' /><div className='text-sm text-gray-400 mr-2'>Back</div>
            </a>
          </Link>
        )}
        <Link href={closeHref || '/'}>
          <a className='flex items-center p-4'>
            <div className='text-sm text-gray-400 mr-2'>Close</div><XIcon className='w-6 h-6 text-white' />
          </a>
        </Link>
      </div>
      <div className='flex-1 flex justify-center items-center'>
        {children}
      </div>
    </div>
  )
}
