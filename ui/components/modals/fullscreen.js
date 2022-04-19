import Link from 'next/link'
import { XIcon, ArrowLeftIcon } from '@heroicons/react/outline'

export default function ({ children, closeHref, backHref }) {
  return (
    <div className='flex flex-col w-full h-full'>
      <div className={`flex flex-none text-right ${backHref ? 'justify-between' : 'justify-end'}`}>
        {backHref && (
          <Link href={backHref || '/'}>
            <a className='flex items-center text-gray-400 px-4'>
              <ArrowLeftIcon className='w-3 h-3 mr-1' /><p className='text-sm'>Back</p>
            </a>
          </Link>
        )}
        <Link href={closeHref || '/'}>
          <a>
            <XIcon className='w-14 h-14 p-4 text-gray-500' />
          </a>
        </Link>
      </div>
      <div className='flex-1 flex justify-center items-center'>
        {children}
      </div>
    </div>
  )
}
