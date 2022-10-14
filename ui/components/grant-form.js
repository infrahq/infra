import { useEffect, useState, useRef } from 'react'
import useSWR from 'swr'
import { Combobox, Listbox } from '@headlessui/react'
import { PlusIcon, CheckIcon, ChevronDownIcon } from '@heroicons/react/outline'
import { usePopper } from 'react-popper'
import * as ReactDOM from 'react-dom'

import { sortByRole } from '../lib/grants'

import RoleSelect from './role-select'

const OPTION_SELECT_ALL = 'select all'

export default function GrantForm({
  grants,
  roles,
  resources,
  multiselect = true,
  onSubmit = () => {},
}) {
  const { data: { items: users } = { items: [] }, mutate: mutateUsers } =
    useSWR('/api/users?limit=1000')
  const { data: { items: groups } = { items: [] }, mutate: mutateGroups } =
    useSWR('/api/groups?limit=1000')

  const [role, setRole] = useState(sortByRole(roles)?.[0])
  const [query, setQuery] = useState('')
  const [selected, setSelected] = useState(null)
  const [options, setOptions] = useState([])

  const [selectedResources, setSelectedResources] = useState([])

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

  const button = useRef()

  useEffect(() => setRole(sortByRole(roles)?.[0]), [roles])

  useEffect(() => setSelectedResources([resources?.[0]]), [resources])

  useEffect(() => {
    if (users && groups) {
      const optionsList = [
        ...(groups?.map(g => ({ ...g, group: true })) || []),
        ...(users?.map(u => ({ ...u, user: true })) || []),
      ]
      const filteredOptions = multiselect
        ? optionsList
        : optionsList?.filter(
            item =>
              !grants?.find(g => g.user === item.id || g.group === item.id)
          )

      setOptions(
        filteredOptions.filter(s =>
          s?.name?.toLowerCase()?.includes(query.toLowerCase())
        )
      )
    }
  }, [users, groups, grants, query])

  return (
    <form
      className='my-2 flex flex-row space-x-3'
      onSubmit={e => {
        e.preventDefault()
        if (resources) {
          console.log(selectedResources)
          onSubmit({
            user: selected.user ? selected.id : undefined,
            group: selected.group ? selected.id : undefined,
            privilege: role,
            selectedResources,
          })
          setRole(sortByRole(roles)?.[0])
          setSelected(null)
          setSelectedResources([resources?.[0]])
        } else {
          onSubmit({
            user: selected.user ? selected.id : undefined,
            group: selected.group ? selected.id : undefined,
            privilege: role,
          })
          setRole(sortByRole(roles)?.[0])
          setSelected(null)
        }
      }}
    >
      <div className='flex flex-1 items-center'>
        <Combobox
          as='div'
          className='relative flex-1'
          value={selected?.name}
          onChange={setSelected}
          onFocus={() => {
            mutateUsers()
            mutateGroups()
          }}
        >
          <Combobox.Input
            className={`block w-full rounded-md border-gray-300 text-xs shadow-sm focus:border-blue-500 focus:ring-blue-500`}
            placeholder='Enter group or user'
            onChange={e => {
              setQuery(e.target.value)
              if (e.target.value.length === 0) {
                setSelected(null)
              }
            }}
            onFocus={() => {
              if (!selected) {
                button.current?.click()
              }
            }}
          />
          {options?.length > 0 && (
            <Combobox.Options className='absolute z-10 mt-2 max-h-60 w-56 origin-top-right divide-y divide-gray-100 overflow-auto rounded-md bg-white text-xs shadow-lg shadow-gray-300/20 ring-1 ring-black ring-opacity-5 focus:outline-none'>
              {options?.map(f => (
                <Combobox.Option
                  key={f.id}
                  value={f}
                  className={({ active }) =>
                    `relative cursor-default select-none py-[7px] px-3 ${
                      active ? 'bg-gray-50' : ''
                    }`
                  }
                >
                  <div className='flex flex-row'>
                    <div className='flex min-w-0 flex-1 flex-col'>
                      <div className='flex justify-between py-0.5 font-medium'>
                        <span className='truncate' title={f.name}>
                          {f.name}
                        </span>
                        {selected && selected.id === f.id && (
                          <CheckIcon
                            data-testid='selected-icon'
                            className='h-3 w-3 stroke-1 text-gray-600'
                            aria-hidden='true'
                          />
                        )}
                      </div>
                      <div className='text-3xs text-gray-500'>
                        {f.user ? 'User' : f.group ? 'Group' : ''}
                      </div>
                    </div>
                  </div>
                </Combobox.Option>
              ))}
            </Combobox.Options>
          )}
          <Combobox.Button className='hidden' ref={button} />
        </Combobox>
      </div>
      {resources?.length > 1 && (
        <div className='relative'>
          <Listbox
            value={selectedResources}
            onChange={v => {
              if (v.includes(OPTION_SELECT_ALL)) {
                if (selectedResources.length !== resources.length) {
                  setSelectedResources([...resources])
                } else {
                  setSelectedResources([resources?.[0]])
                }
                return
              }

              setSelectedResources(v)
            }}
            multiple
          >
            <div className='relative'>
              <Listbox.Button
                ref={setReferenceElement}
                className='relative w-48 cursor-default rounded-md border border-gray-300 bg-white py-2 pl-3 pr-8 text-left text-xs shadow-sm hover:cursor-pointer hover:bg-gray-100 focus:outline-none'
              >
                <div className='flex space-x-1 truncate'>
                  <span className='pointer-events-none absolute inset-y-0 right-0 flex items-center pr-2'>
                    <ChevronDownIcon
                      className='h-4 w-4 stroke-1 text-gray-700'
                      aria-hidden='true'
                    />
                  </span>
                  <span className='text-gray-700'>
                    {selectedResources.length > 0
                      ? selectedResources[0]
                      : '[Namespaces]'}
                  </span>
                  {selectedResources.length - 1 > 0 && (
                    <span> + {selectedResources.length - 1}</span>
                  )}
                </div>
              </Listbox.Button>
              {ReactDOM.createPortal(
                <Listbox.Options
                  ref={setPopperElement}
                  style={styles.popper}
                  {...attributes.popper}
                  className='absolute z-[8] w-48 overflow-auto rounded-md border  border-gray-200 bg-white text-left text-xs text-gray-800 shadow-lg shadow-gray-300/20 focus:outline-none'
                >
                  <div className='max-h-64 overflow-auto'>
                    {resources?.map(r => (
                      <Listbox.Option
                        key={r}
                        className={({ active }) =>
                          `${
                            active ? 'bg-gray-100' : ''
                          } select-none py-2 px-3 hover:cursor-pointer`
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
                            </div>
                          </div>
                        )}
                      </Listbox.Option>
                    ))}
                  </div>
                  {resources.length > 1 && (
                    <Listbox.Option
                      className={({ active }) =>
                        `${
                          active ? 'bg-gray-50' : 'bg-white'
                        } group flex w-full items-center border-t border-gray-100 px-2 py-1.5 text-xs font-medium text-blue-500 hover:cursor-pointer`
                      }
                      value={OPTION_SELECT_ALL}
                    >
                      <div className='flex flex-row items-center py-0.5'>
                        {selectedResources.length !== resources.length
                          ? 'Select all'
                          : 'Reset'}
                      </div>
                    </Listbox.Option>
                  )}
                </Listbox.Options>,
                document.querySelector('body')
              )}
            </div>
          </Listbox>
        </div>
      )}
      {roles?.length > 1 && (
        <div className='relative'>
          <RoleSelect onChange={setRole} role={role} roles={roles} />
        </div>
      )}
      <div className='relative'>
        <button
          disabled={
            resources ? selectedResources.length === 0 || !selected : !selected
          }
          type='submit'
          className='inline-flex items-center rounded-md border border-transparent bg-black px-4 py-[7px] text-xs font-medium text-white shadow-sm hover:cursor-pointer hover:bg-gray-800 disabled:cursor-not-allowed disabled:opacity-30'
        >
          <PlusIcon className='mr-1 h-3 w-3' />
          Add
        </button>
      </div>
    </form>
  )
}
