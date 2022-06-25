import useSWR from 'swr'
import { Listbox } from '@headlessui/react'
import { CheckIcon, ChevronDownIcon } from '@heroicons/react/solid'
import { XIcon } from '@heroicons/react/outline'

import { sortByPrivilege } from '../lib/grants'

const OPTION_REMOVE = 'remove'

const descriptions = {
  'cluster-admin': 'Super-user access to perform any action on any resource',
  admin: 'Read and write access to all resources',
  edit: 'Read and write access to most resources, but not roles',
  view: 'Read-only access to see most resources',
  logs: 'Read and stream logs',
  exec: 'Shell to a running container',
  'port-forward': 'Use port-forwarding to access applications'
}

export default function ({
  resource,
  role,
  roles,
  onChange,
  onRemove,
  remove,
  direction = 'right'
}) {
  const parts = resource?.split('.') || []
  const hasParent = parts?.length > 1

  const { data: { items } = {} } = useSWR(() => resource && `/api/destinations?name=${hasParent ? parts[0] : resource}`)
  roles = roles || items?.[0]?.roles || []
  roles = roles?.sort(sortByPrivilege)?.filter(r => hasParent ? r !== 'cluster-admin' : true)

  return (
    <Listbox
      value={role}
      onChange={v => v === OPTION_REMOVE ? onRemove() : onChange(v)}
    >
      <div className='relative'>
        <Listbox.Button className='bg-black relative w-32 pl-3 pr-8 py-2 text-left cursor-default focus:outline-none text-2xs'>
          <span className='absolute inset-y-0 right-0 flex items-center pr-2 pointer-events-none'>
            <ChevronDownIcon className='h-4 w-4 stroke-1 text-gray-400' aria-hidden='true' />
          </span>
          <span className='block truncate text-gray-400'>{role}</span>
        </Listbox.Button>
        <Listbox.Options className={`absolute z-10 w-48 ${direction === 'right' ? '' : 'right-0'} text-white text-2xs mt-2 bg-gray-800 border border-gray-700 rounded-md ring-1 ring-black ring-opacity-5 overflow-auto focus:outline-none`}>
          <div className={`overflow-scroll max-h-64 ${remove ? 'mb-9' : ''}`}>
            {roles?.map(r => (
              <Listbox.Option
                key={r}
                className={({ active }) => `${active ? 'bg-gray-700' : ''} cursor-default select-none relative py-2 px-3`}
                value={r}
              >
                {({ selected }) => (
                  <div className='flex flex-row'>
                    <div className='flex-1 flex flex-col'>
                      <div className='font-medium flex justify-between py-0.5'>
                        {r}
                        {selected && <CheckIcon className='h-3 w-3 stroke-1' aria-hidden='true' />}
                      </div>
                      <div className='text-3xs text-gray-400'>{descriptions[r]}</div>
                    </div>
                  </div>
                )}
              </Listbox.Option>
            ))}
          </div>
          {remove && (
            <Listbox.Option
              className={({ active }) => `${active ? 'bg-gray-700' : ''} cursor-default select-none py-2 px-3 left-0 right-0 absolute bottom-0 border-t border-gray-700 hover:bg-gray-700 z-10`}
              value={OPTION_REMOVE}
            >
              <div className='flex flex-row items-center py-0.5'>
                <XIcon className='w-3 h-3 mr-2' />
                Remove
              </div>
            </Listbox.Option>
          )}
        </Listbox.Options>
      </div>
    </Listbox>
  )
}
