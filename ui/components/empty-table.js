import Link from 'next/link'
import { PlusIcon } from '@heroicons/react/outline'

export default function EmptyTable({
  title,
  subtitle,
  iconPath,
  buttonText,
  buttonHref,
}) {
  return (
    <div className='mx-auto mb-20 flex flex-1 flex-col justify-center text-center'>
      <img src={iconPath} className='mx-auto my-4 h-7 w-7' alt={title} />
      <h1 className='mb-2 text-base font-bold'>{title}</h1>
      <h2 className='mx-auto mb-4 max-w-xs text-2xs text-gray-300'>
        {subtitle}
      </h2>
      {buttonHref && (
        <Link href={buttonHref}>
          <button className='my-2 mx-auto flex items-center rounded-md border border-violet-300 px-4 py-2.5'>
            <PlusIcon className='mr-1 h-2.5 w-2.5' />
            <div className='text-2xs font-medium text-violet-100'>
              {buttonText}
            </div>
          </button>
        </Link>
      )}
    </div>
  )
}
