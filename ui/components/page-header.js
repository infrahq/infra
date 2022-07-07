import Link from 'next/link'
import { PlusIcon } from '@heroicons/react/outline'

export default function PageHeader({ header, buttonLabel, buttonHref }) {
  return (
    <div className='z-10 flex min-h-[40px] flex-none items-center justify-between bg-black py-3 px-6'>
      <h1 className='py-3 text-xs font-semibold'>{header}</h1>
      {buttonHref && (
        <Link href={buttonHref}>
          <button className='flex items-center rounded-md  border border-violet-300 px-4 py-3 text-sm text-violet-100'>
            <PlusIcon className='mr-1 h-3 w-3' />
            <div className='text-2xs font-medium leading-none'>
              {buttonLabel}
            </div>
          </button>
        </Link>
      )}
    </div>
  )
}
