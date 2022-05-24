import Link from 'next/link'
import { PlusIcon } from '@heroicons/react/outline'

export default function ({ header, buttonLabel, buttonHref }) {
  return (
    <div className='flex justify-between items-center my-3 min-h-[40px]'>
      <h1 className='text-xs font-semibold'>{header}</h1>
      {buttonHref && (
        <Link href={buttonHref}>
          <button className='border border-violet-300 text-violet-100  rounded-md flex items-center text-sm px-4 py-2.5'>
            <PlusIcon className='w-3 h-3 mr-1' />
            <div className='font-medium text-2xs'>
              {buttonLabel}
            </div>
          </button>
        </Link>
      )}
    </div>
  )
}
