import Link from 'next/link'
import { PlusIcon } from '@heroicons/react/outline'

export default function PageHeader({ header, buttonLabel, buttonHref }) {
  return (
    <div className='flex min-h-[40px] flex-none items-center justify-between py-3 px-6 xl:px-0'>
      <h1 className='text-md font-semibold text-gray-900'>{header}</h1>
      {buttonHref && (
        <Link href={buttonHref} data-testid='page-header-button-link'>
          <button className='inline-flex items-center rounded-md border border-transparent bg-black px-4 py-2 text-sm font-medium text-white shadow-sm hover:bg-gray-800'>
            <PlusIcon className='mr-1 h-3 w-3' />
            <div className='text-2xs leading-none'>{buttonLabel}</div>
          </button>
        </Link>
      )}
    </div>
  )
}
