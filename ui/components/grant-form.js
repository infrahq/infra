import { useEffect, useState, useRef } from 'react'
import useSWR from 'swr'
import { Combobox } from '@headlessui/react'
import { PlusIcon } from '@heroicons/react/outline'

import RoleSelect from './role-select'
import ComboboxItem from './combobox-item'

export default function GrantForm({ roles, onSubmit = () => {} }) {
  const { data: { items: users } = { items: [] }, mutate: mutateUsers } =
    useSWR('/api/users?limit=1000')
  const { data: { items: groups } = { items: [] }, mutate: mutateGroups } =
    useSWR('/api/groups?limit=1000')

  const [role, setRole] = useState(roles?.[0])
  const [query, setQuery] = useState('')
  const [selected, setSelected] = useState(null)
  const button = useRef()

  useEffect(() => setRole(roles?.[0]), [roles])

  const filtered = [
    ...users.map(u => ({ ...u, user: true })),
    ...groups.map(g => ({ ...g, group: true })),
  ].filter(s => s?.name?.toLowerCase()?.includes(query.toLowerCase()))

  return (
    <form
      className='my-2 flex flex-row'
      onSubmit={e => {
        e.preventDefault()
        onSubmit({
          user: selected.user ? selected.id : undefined,
          group: selected.group ? selected.id : undefined,
          privilege: role,
        })
        setRole(roles?.[0])
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
            placeholder='User or group'
            onChange={e => setQuery(e.target.value)}
            onFocus={() => {
              if (!selected) {
                button.current?.click()
              }
            }}
          />
          {filtered.length > 0 && (
            <Combobox.Options className='absolute -left-[13px] z-10 mt-1 max-h-60 w-56 overflow-auto rounded-md border border-gray-700 bg-gray-800 py-1 text-2xs ring-1 ring-black ring-opacity-5 focus:outline-none'>
              {filtered?.map(f => (
                <Combobox.Option
                  key={f.id}
                  value={f}
                  className={({ active }) =>
                    `relative cursor-default select-none py-[7px] px-3 text-gray-200 ${
                      active ? 'bg-gray-700' : ''
                    }`
                  }
                >
                  <ComboboxItem
                    title={f.name}
                    subtitle={f.user ? 'User' : f.group ? 'Group' : ''}
                    selected={selected && selected.id === f.id}
                  />
                </Combobox.Option>
              ))}
            </Combobox.Options>
          )}
          <Combobox.Button className='hidden' ref={button} />
        </Combobox>
      </div>
      {roles?.length > 1 && (
        <div className='relative px-2'>
          <RoleSelect onChange={setRole} role={role} roles={roles} />
        </div>
      )}
      <div className='relative'>
        <button
          disabled={!selected}
          type='submit'
          className='inline-flex items-center rounded-md border border-transparent bg-black px-4 py-[7px] text-xs font-medium text-white shadow-sm hover:bg-gray-800'
        >
          <PlusIcon className='mr-1 h-3 w-3' />
          Add
        </button>
      </div>
    </form>
  )
}
