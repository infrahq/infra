import { useEffect, useState, useRef } from 'react'
import useSWR from 'swr'
import { Combobox } from '@headlessui/react'
import { CheckIcon } from '@heroicons/react/solid'
import { PlusIcon } from '@heroicons/react/outline'

import RoleDropdown from './role-dropdown'

export default function ({ roles, onSubmit = () => {} }) {
  const { data: { items: users } = { items: [] } } = useSWR('/api/users')
  const { data: { items: groups } = { items: [] } } = useSWR('/api/groups')

  const [role, setRole] = useState(roles?.[0])
  const [query, setQuery] = useState('')
  const [selected, setSelected] = useState(null)
  const button = useRef()

  useEffect(() => setRole(roles?.[0]), [roles])

  const filtered = [
    ...users.map(u => ({ ...u, user: true })),
    ...groups.map(g => ({ ...g, group: true }))
  ]
    .filter(s => s?.name?.toLowerCase()?.includes(query.toLowerCase()))
    .filter(s => s.name !== 'connector')

  return (
    <form
      onSubmit={e => {
        e.preventDefault()
        onSubmit({
          user: selected.user ? selected.id : undefined,
          group: selected.group ? selected.id : undefined,
          privilege: role
        })
      }}
      className='flex my-2'
    >
      <div className='flex items-center flex-1 border-b border-gray-800'>
        <Combobox
          as='div'
          className='relative flex-1'
          value={selected?.name}
          onChange={setSelected}
        >
          <Combobox.Input
            className='relative placeholder:italic text-xs w-full pr-2 py-3 bg-transparent focus:outline-none disabled:opacity-30'
            placeholder='User or group'
            onChange={e => setQuery(e.target.value)}
            onFocus={() => {
              if (!selected) {
                button.current?.click()
              }
            }}
          />
          {filtered.length > 0 && (
            <Combobox.Options
              className='absolute z-10 -left-[13px] mt-1 max-h-60 w-56 overflow-auto rounded-md bg-gray-800 border border-gray-700 py-1 text-2xs ring-1 ring-black ring-opacity-5 focus:outline-none'
            >
              {filtered?.map(f => (
                <Combobox.Option
                  key={f.id}
                  value={f}
                  className={({ active }) => `relative cursor-default select-none py-2 px-3 hover:bg-gray-700 ${active ? 'bg-gray-700' : ''}`}
                >
                  <div className='flex flex-row'>
                    <div className='flex-1 min-w-0 flex flex-col'>
                      <div className='font-medium flex justify-between py-0.5'>
                        <span className='truncate' title={f.name}>{f.name}</span>
                        {f.id === selected?.id && <CheckIcon className='h-3 w-3 stroke-1' aria-hidden='true' />}
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
          <Combobox.Button className='hidden' ref={button} />
        </Combobox>
        {roles?.length > 1 && (
          <RoleDropdown onChange={setRole} role={role} roles={roles} />
        )}
      </div>
      <button
        disabled={!selected}
        type='submit'
        className='flex items-center border border-violet-300 disabled:opacity-30 disabled:transform-none disabled:transition-none cursor-pointer disabled:cursor-default sm:ml-4 sm:mt-0 rounded-md text-2xs px-3 py-3'
      >
        <PlusIcon className='w-3 h-3 mr-1.5' />
        <div className='text-violet-100'>
          Add
        </div>
      </button>
    </form>
  )
}
