import useSWR from 'swr'
import { Listbox } from '@headlessui/react'
import { CheckIcon, ChevronDownIcon } from '@heroicons/react/solid'
import { XIcon } from '@heroicons/react/outline'

import {
  sortByPrivilege,
  descriptions,
  sortByHasDescription,
} from '../lib/grants'

const OPTION_REMOVE = 'remove'

export default function RoleSelect({
  resource,
  role,
  roles,
  onChange,
  onRemove,
  remove,
  direction = 'right',
}) {
  const parts = resource?.split('.') || []
  const hasParent = parts?.length > 1

  const { data: { items } = {} } = useSWR(
    () =>
      resource && `/api/destinations?name=${hasParent ? parts[0] : resource}`
  )

  roles = roles || items?.[0]?.roles || []
  roles = roles
    ?.sort(sortByPrivilege)
    ?.sort(sortByHasDescription)
    ?.filter(r => !hasParent || r !== 'cluster-admin')

  return (
    <Listbox
      value={role}
      onChange={v => {
        if (v === role) {
          return
        }

        if (v === OPTION_REMOVE) {
          onRemove()
          return
        }

        onChange(v)
      }}
    >
      <div className='relative'>
        <Listbox.Button className='relative w-32 cursor-default bg-black py-2 pl-3 pr-8 text-left text-2xs focus:outline-none'>
          <span className='pointer-events-none absolute inset-y-0 right-0 flex items-center pr-2'>
            <ChevronDownIcon
              className='h-4 w-4 stroke-1 text-gray-400'
              aria-hidden='true'
            />
          </span>
          <span className='block truncate text-gray-400'>{role}</span>
        </Listbox.Button>
        <Listbox.Options
          className={`absolute z-10 w-48 ${
            direction === 'right' ? '' : 'right-0'
          } mt-2 overflow-auto rounded-md border border-gray-700 bg-gray-800 text-2xs text-white ring-1 ring-black ring-opacity-5 focus:outline-none`}
        >
          <div className={`max-h-64 overflow-auto ${remove ? 'mb-9' : ''}`}>
            {roles?.map(r => (
              <Listbox.Option
                key={r}
                className={({ active }) =>
                  `${
                    active ? 'bg-gray-700' : ''
                  } relative cursor-default select-none py-2 px-3`
                }
                value={r}
              >
                {({ selected }) => (
                  <div className='flex flex-row'>
                    <div className='flex flex-1 flex-col'>
                      <div className='flex justify-between py-0.5 font-medium'>
                        {r}
                        {selected && (
                          <CheckIcon
                            className='h-3 w-3 stroke-1'
                            aria-hidden='true'
                          />
                        )}
                      </div>
                      <div className='text-3xs text-gray-400'>
                        {descriptions[r]}
                      </div>
                    </div>
                  </div>
                )}
              </Listbox.Option>
            ))}
          </div>
          {remove && (
            <Listbox.Option
              className={({ active }) =>
                `${
                  active ? 'bg-gray-700' : ''
                } absolute left-0 right-0 bottom-0 z-10 cursor-default select-none border-t border-gray-700 py-2 px-3 hover:bg-gray-700`
              }
              value={OPTION_REMOVE}
            >
              <div className='flex flex-row items-center py-0.5'>
                <XIcon className='mr-2 h-3 w-3' />
                Remove
              </div>
            </Listbox.Option>
          )}
        </Listbox.Options>
      </div>
    </Listbox>
  )
}
