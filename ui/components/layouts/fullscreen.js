import Link from 'next/link'
import { XIcon, ChevronLeftIcon } from '@heroicons/react/outline'

import AuthRequired from '../auth-required'

function Layout({ children, backHref, closeHref }) {
  return (
    <div className='flex h-full w-full flex-col'>
      <div
        className={`flex flex-none justify-end text-right ${
          backHref ? 'justify-between' : 'justify-end'
        }`}
      >
        {backHref && (
          <Link href={backHref || '/'}>
            <a className='flex items-center p-4 text-3xs uppercase text-gray-400 hover:text-black'>
              <ChevronLeftIcon className='mr-1.5 h-5 w-5 stroke-1' />
              Back
            </a>
          </Link>
        )}
        <Link href={closeHref || '/'}>
          <a className='flex items-center p-4 text-3xs uppercase text-gray-400 hover:text-black'>
            Close
            <XIcon className='ml-1 h-6 w-6 stroke-1' />
          </a>
        </Link>
      </div>
      <div className='mb-10 flex flex-1 items-center justify-center'>
        <div className='w-full max-w-xs rounded-lg border border-gray-800'>
          {children}
        </div>
      </div>
    </div>
  )
}

export default function Fullscreen(props) {
  return (
    <AuthRequired>
      <Layout {...props} />
    </AuthRequired>
  )
}
