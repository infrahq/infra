import useSWR from 'swr'
import { Listbox } from '@headlessui/react'
import { CheckIcon, ChevronDownIcon } from '@heroicons/react/solid'
import { XIcon } from '@heroicons/react/outline'
import { useState } from 'react'
import { usePopper } from 'react-popper'
import * as ReactDOM from 'react-dom'

import { descriptions, sortByRole } from '../lib/grants'

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

  const [referenceElement, setReferenceElement] = useState(null)
  const [popperElement, setPopperElement] = useState(null)
  let { styles, attributes } = usePopper(referenceElement, popperElement, {
    placement: 'bottom-end',
    modifiers: [
      {
        name: 'flip',
        enabled: false,
      },
    ],
  })

  roles = roles || items?.[0]?.roles || []
  roles = sortByRole(roles).filter(r => !hasParent || r !== 'cluster-admin')

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
        <Listbox.Button
          ref={setReferenceElement}
          className='relative w-36 cursor-default rounded-md border border-gray-300 bg-white py-2 pl-3 pr-8 text-left text-xs shadow-sm hover:cursor-pointer hover:bg-gray-100 focus:outline-none'
        >
          <span className='pointer-events-none absolute inset-y-0 right-0 flex items-center pr-2'>
            <ChevronDownIcon
              className='h-4 w-4 stroke-1 text-gray-700'
              aria-hidden='true'
            />
          </span>
          <span className='block truncate text-gray-700'>{role}</span>
        </Listbox.Button>
        {ReactDOM.createPortal(
          <Listbox.Options
            ref={setPopperElement}
            style={styles.popper}
            {...attributes.popper}
            className={`absolute z-[8] w-48 ${
              direction === 'right' ? '' : 'right-0'
            } mt-2 overflow-auto rounded-md border  border-gray-200 bg-white text-left text-xs text-gray-800 shadow-lg shadow-gray-300/20 focus:outline-none`}
          >
            <div className='max-h-64 overflow-auto'>
              {roles?.map(r => (
                <Listbox.Option
                  key={r}
                  className={({ active }) =>
                    `${
                      active ? 'bg-gray-100' : ''
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
                              className='h-3 w-3 stroke-1 text-gray-600'
                              aria-hidden='true'
                            />
                          )}
                        </div>
                        <div className='text-3xs text-gray-600'>
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
                    active ? 'bg-gray-50' : 'bg-white'
                  } group flex w-full items-center border-t border-gray-100 px-2 py-1.5 text-xs font-medium text-red-500`
                }
                value={OPTION_REMOVE}
              >
                <div className='flex flex-row items-center py-0.5'>
                  <XIcon className='mr-1 mt-px h-3.5 w-3.5' /> Remove
                </div>
              </Listbox.Option>
            )}
          </Listbox.Options>,
          document.querySelector('body')
        )}
      </div>
    </Listbox>
  )
}
