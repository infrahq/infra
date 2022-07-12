import { CheckIcon } from '@heroicons/react/solid'

export default function ListRow({ item, selected }) {
  return (
    <div className='flex flex-row'>
      <div className='flex min-w-0 flex-1 flex-col'>
        <div className='flex justify-between py-0.5 font-medium'>
          <span className='truncate' title={item.name}>
            {item.name}
          </span>
          {selected && item.id === selected?.id && (
            <CheckIcon className='h-3 w-3 stroke-1' aria-hidden='true' />
          )}
        </div>
        <div className='text-3xs text-gray-400'>
          {item.user && 'User'}
          {item.group && 'Group'}
        </div>
      </div>
    </div>
  )
}
