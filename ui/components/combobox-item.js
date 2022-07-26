import { CheckIcon } from '@heroicons/react/solid'

export default function ComboboxItem({ title, subtitle, selected = false }) {
  return (
    <div className='flex flex-row'>
      <div className='flex min-w-0 flex-1 flex-col'>
        <div className='flex justify-between py-0.5 font-medium'>
          <span className='truncate' title={title}>
            {title}
          </span>
          {selected && (
            <CheckIcon
              data-testid='selected-icon'
              className='h-3 w-3 stroke-1'
              aria-hidden='true'
            />
          )}
        </div>
        <div className='text-3xs text-gray-400'>{subtitle}</div>
      </div>
    </div>
  )
}
