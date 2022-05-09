import Link from 'next/link'
import { PlusIcon } from '@heroicons/react/outline'

export default function ({header, buttonLabel, buttonHref}) {
  return  (
    <div className='flex justify-between items-center'>
      <h1 className='text-title font-bold'>{header}</h1>
      {buttonHref && (
        <Link href={buttonHref}>
          <button className='bg-gradient-to-tr from-indigo-300 to-pink-100 hover:from-indigo-200 hover:to-pink-50 rounded-md p-0.5 my-2'>
            <div className='bg-black rounded-md flex items-center text-sm px-4 py-1.5'>
              <PlusIcon className='w-2 h-2 mr-1' />
              <div className='text-purple-50'>
                {buttonLabel}
              </div>
            </div>
          </button>
        </Link>
      )}
    </div>
  )
}