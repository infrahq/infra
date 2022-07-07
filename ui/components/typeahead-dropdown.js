import { Combobox } from '@headlessui/react'
import { CheckIcon } from '@heroicons/react/solid'

export default function TypeaheadDropdown({ filtered, selected }) {
  return (
    <>
      {filtered.length > 0 && (
        <Combobox.Options className='absolute -left-[13px] z-10 mt-1 max-h-60 w-56 overflow-auto rounded-md border border-gray-700 bg-gray-800 py-1 text-2xs ring-1 ring-black ring-opacity-5 focus:outline-none'>
          {filtered?.map(f => (
            <Combobox.Option
              key={f.id}
              value={f}
              className={({ active }) =>
                `relative cursor-default select-none py-2 px-3 hover:bg-gray-700 ${
                  active ? 'bg-gray-700' : ''
                }`
              }
            >
              <div className='flex flex-row'>
                <div className='flex min-w-0 flex-1 flex-col'>
                  <div className='flex justify-between py-0.5 font-medium'>
                    <span className='truncate' title={f.name}>
                      {f.name}
                    </span>
                    {selected && f.id === selected?.id && (
                      <CheckIcon
                        className='h-3 w-3 stroke-1'
                        aria-hidden='true'
                      />
                    )}
                  </div>
                  <div className='text-3xs text-gray-400'>
                    {f.user && 'User'}
                    {f.group && 'Group'}
                  </div>
                </div>
              </div>
            </Combobox.Option>
          ))}
        </Combobox.Options>
      )}
    </>
  )
}
