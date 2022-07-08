import { XIcon } from '@heroicons/react/outline'

export default function Badge({ children, onRemove }) {
  return (
    <div className='m-1 flex items-center justify-center overflow-hidden text-ellipsis rounded-md bg-gray-800 py-1 px-2 font-medium text-white'>
      <div className='max-w-full flex-initial overflow-hidden text-ellipsis text-xs font-normal leading-none'>
        {children}
      </div>
      <div className='flex flex-auto flex-row-reverse pl-1'>
        <XIcon
          className='h-2 w-2 hover:cursor-pointer'
          aria-hidden='true'
          onClick={onRemove}
        />
      </div>
    </div>
  )
}
