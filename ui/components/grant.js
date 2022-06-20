import useSWR, { useSWRConfig } from 'swr'
import { useState } from 'react'
import { PlusIcon } from '@heroicons/react/outline'

import InputDropdown from './input'
import ErrorMessage from './error-message'

function User ({ id }) {
  if (!id) {
    return null
  }

  const { data: user } = useSWR(`/api/users/${id}`, { fallbackData: { name: '' } })

  return (
    <p className='text-2xs'>{user.name}</p>
  )
}

export default function ({ resource = '' }) {
  // fetch grants for resource, and for parent resources
  // todo: support arbitrary resource depth
  const parts = resource.split('.')

  const { data: { items } = {} } = useSWR(() => `/api/grants?resource=${resource}`)
  const { data: { items: inherited } = {} } = useSWR(() => parts.length > 1 ? `/api/grants?resource=${parts[0]}` : null)
  const { mutate } = useSWRConfig()
  const [name, setName] = useState('')
  const [error, setError] = useState('')
  const [role, setRole] = useState('view')
  const options = ['view', 'edit', 'admin', 'remove']

  async function grant (user, privilege) {
    mutate(`/api/grants?resource=${resource}`, async ({ items: grants } = { items: [] }) => {
      const res = await fetch('/api/grants', {
        method: 'POST',
        body: JSON.stringify({ user, resource, privilege })
      })

      if (!res.ok) {
        setError('could not create grant')
        return { items }
      }

      const data = await res.json()

      // replace any existing grants
      const existing = grants.filter(g => g.user === user)
      for (const e of existing) {
        await fetch(`/api/grants/${e.id}`, { method: 'DELETE' })
      }

      return { items: [...grants.filter(grant => grant?.user !== user), data] }
    })
  }

  async function remove (id) {
    mutate(`/api/grants?resource=${resource}`, async ({ items: grants } = { items: [] }) => {
      await fetch(`/api/grants/${id}`, { method: 'DELETE' })
      return { items: grants?.filter(item => item?.id !== id) }
    }, { optimisticData: { items: items?.filter(item => item?.id !== id) } })
  }

  return (
    <>
      <form
        onSubmit={async e => {
          e.preventDefault()
          const res = await fetch(`/api/users?name=${name}`)
          if (!res.ok) {
            setError(res)
            return
          }

          const users = await res.json()
          if (!users?.items?.length) {
            setError('User does not exist')
            return
          }

          await grant(users?.items?.[0]?.id, role)

          setName('')
          setRole('view')
        }}
        className={`flex gap-1 mt-3 ${error ? 'mb-2' : 'mb-4'}`}
      >
        <div className='flex-1'>
          <InputDropdown
            value={name}
            placeholder='Username'
            error={error}
            optionType='role'
            options={options.filter(item => item !== 'remove')}
            handleInputChange={e => {
              setName(e.target.value)
              setError('')
            }}
            handleSelectOption={e => setRole(e.target.value)}
            selectedItem={role}
          />
        </div>
        <button
          disabled={name.length === 0}
          type='submit'
          className='flex items-center border border-violet-300 disabled:opacity-30 disabled:transform-none disabled:transition-none cursor-pointer disabled:cursor-default mt-4 mr-auto sm:ml-4 sm:mt-0 rounded-md text-2xs px-3 py-3'
        >
          <PlusIcon className='w-3 h-3 mr-1.5' />
          <div className='text-violet-100'>
            Share
          </div>
        </button>
      </form>
      {error && <ErrorMessage message={error} />}
      <div className='py-2 overflow-y-auto max-h-screen'>
        {items?.length > 0 && items?.sort((a, b) => (a.user).localeCompare(b.user)).map(item => (
          <div key={item.id} className='flex justify-between items-center'>
            <User id={item.user} />
            <select
              id='role'
              name='role'
              className='ml-auto pl-3 pr-1 py-2 border-gray-300 focus:outline-none text-2xs text-gray-400 bg-transparent'
              defaultValue={item.privilege}
              onChange={e => {
                if (e.target.value === 'remove') {
                  remove(item.id)
                  return
                }

                grant(item.user, e.target.value)
              }}
            >
              {options.map((option) => (
                <option key={option} value={option}>{option}</option>
              ))}
            </select>
          </div>
        ))}
        {inherited?.length > 0 && inherited?.sort((a, b) => (a.user).localeCompare(b.user)).map(item => (
          <div key={item.id} className='grid grid-cols-7'>
            <div className='col-span-4 py-2'><User id={item.user} /></div>
            <div
              title='This access is inherited by cluster-level access and cannot be edited here'
              className='col-span-2 my-2 text-2xs text-gray-400 border rounded px-2 mx-auto bg-gray-800 border-gray-800'
            >
              cluster access
            </div>
            <div className='py-2 text-2xs text-gray-400'>
              {item.privilege}
            </div>
          </div>
        ))}
      </div>
      {!inherited?.length && !items?.length && (
        <div className='text-2xs text-gray-400 italic w-2/3'>
          Share access by inviting your team and assigning their roles.
        </div>
      )}
    </>
  )
}
