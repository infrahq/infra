import { Listbox } from '@headlessui/react'
import { CheckIcon, ChevronDownIcon } from '@heroicons/react/solid'
import { useState } from 'react'
import { usePopper } from 'react-popper'
import * as ReactDOM from 'react-dom'

import { descriptions } from '../lib/grants'

export default function RoleSelect({ role, roles, onChange }) {
  const [referenceElement, setReferenceElement] = useState(null)
  const [popperElement, setPopperElement] = useState(null)
  let { styles, attributes } = usePopper(referenceElement, popperElement, {
    placement: 'bottom-end',
    modifiers: [
      {
        name: 'flip',
        enabled: false,
      },
      {
        name: 'offset',
        options: {
          offset: [0, 5],
        },
      },
    ],
  })

  return (
    <Listbox
      value={role}
      onChange={v => {
        if (v === role) {
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
            className='absolute z-[8] w-48 overflow-auto rounded-md border  border-gray-200 bg-white text-left text-xs text-gray-800 shadow-lg shadow-gray-300/20 focus:outline-none'
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
          </Listbox.Options>,
          document.querySelector('body')
        )}
      </div>
    </Listbox>
  )
}
