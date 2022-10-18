import { useEffect, useState, useRef } from 'react'
import useSWR from 'swr'
import { Combobox } from '@headlessui/react'
import { PlusIcon, CheckIcon } from '@heroicons/react/outline'

import { sortByRole } from '../lib/grants'

import RoleSelect from './role-select'

export default function GrantForm({
  grants,
  roles,
  selectedResources,
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

  const button = useRef()

  useEffect(
    () =>
      setRole(
        selectedResources?.length > 0
          ? sortByRole(roles)?.filter(r => r != 'cluster-admin')[0]
          : sortByRole(roles)?.[0]
      ),
    [roles, selectedResources]
  )

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
        onSubmit({
          user: selected.user ? selected.id : undefined,
          group: selected.group ? selected.id : undefined,
          privilege: role,
          selectedResources,
        })

        setRole(sortByRole(roles)?.[0])
        setSelected(null)
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
      {roles?.length > 1 && (
        <div className='relative'>
          <RoleSelect
            onChange={setRole}
            role={role}
            roles={
              selectedResources.length > 0
                ? roles.filter(r => r != 'cluster-admin')
                : roles
            }
          />
        </div>
      )}
      <div className='relative'>
        <button
          disabled={!selected}
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
