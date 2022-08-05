import { XIcon } from '@heroicons/react/outline'

export default function Badge({ children, onRemove }) {
  return (
    <div className='my-1 mr-1 flex items-center justify-center overflow-hidden text-ellipsis rounded-md bg-gray-800 font-medium text-white first:my-0'>
      <div className='max-w-full flex-initial overflow-hidden text-ellipsis py-1 pl-2 text-xs font-normal leading-none'>
        {children}
      </div>
      <div
        className='group flex flex-auto flex-row-reverse py-1 px-2 hover:cursor-pointer'
        onClick={onRemove}
      >
        <XIcon
          className='h-3 w-3 text-gray-400 group-hover:text-white'
          data-testid='badge-remove-icon'
          aria-hidden='true'
        />
      </div>
    </div>
  )
}
