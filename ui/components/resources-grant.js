import { useState } from 'react'
import useSWR, { mutate } from 'swr'
import { PlusIcon } from '@heroicons/react/outline'

import InputDropdown from '../components/input'
import ErrorMessage from '../components/error-message'

export default function ({ id }) {
  const { data: { items: grants } = { items: [] } } = useSWR(`/api/grants?user=${id}`)
  const { data: { items: destinations } = { items: [] } } = useSWR('/api/destinations')

  const [infrastructure, setInfrastructure] = useState('')
  const [error, setError] = useState('')
  const [role, setRole] = useState('view')

  const options = ['view', 'edit', 'admin', 'remove']

  function handleInputChang (value) {
    setInfrastructure(value)
    setError('')
  }

  function handleKeyDownEvent (key) {
    if (key === 'Enter' && infrastructure.length > 0) {
      handleShareGrant()
    }
  }

  async function handleShareGrant () {
    if (destinations.find(d => d.name === infrastructure)) {
      grant(id)
    } else {
      setError('Infrastructure does not exist')
    }
  }

  function grant (user, privilege = role, resource = infrastructure, exist = false, deleteGrantId) {
    mutate(`/api/grants?user=${id}`, async ({ items: grantsList = [] }) => {
      const res = await fetch('/api/grants', {
        method: 'POST',
        body: JSON.stringify({ user, resource, privilege })
      })

      const data = await res.json()

      setInfrastructure('')

      if (exist) {
        await fetch(`/api/grants/${deleteGrantId}`, { method: 'DELETE' })
        return { items: [...grantsList.filter(grant => grant?.user !== user), data] }
      }

      return { items: [...grantsList, data] }
    })
  }

  function handleUpdateGrant (privilege, resource, grantId, user) {
    if (privilege !== 'remove') {
      return grant(user, privilege, resource, true, grantId)
    }

    mutate(`/api/grants?user=${user}`, async ({ items: userGrantsList = [] }) => {
      await fetch(`/api/grants/${grantId}`, { method: 'DELETE' })
      return { items: userGrantsList?.filter(item => item?.id !== grantId) }
    }, { optimisticData: { items: grants?.filter(g => g?.id !== grantId) } })
  }

  return (
    <>
      <div className={`flex gap-1 mt-3 ${error ? 'mb-2' : 'mb-4'}`}>
        <div className='flex-1'>
          <InputDropdown
            type='text'
            value={infrastructure}
            placeholder='Infrastructure, cluster'
            error={error}
            optionType='role'
            options={options.filter((item) => item !== 'remove')}
            handleInputChange={e => handleInputChang(e.target.value)}
            handleSelectOption={e => setRole(e.target.value)}
            handleKeyDown={(e) => handleKeyDownEvent(e.key)}
            selectedItem={role}
          />
        </div>
        <button
          onClick={() => handleShareGrant()}
          disabled={infrastructure.length === 0}
          type='button'
          className='flex items-center border border-violet-300 disabled:opacity-30 disabled:transform-none disabled:transition-none cursor-pointer disabled:cursor-default mt-4 mr-auto sm:ml-4 sm:mt-0 rounded-md text-2xs px-3 py-3'
        >
          <PlusIcon className='w-3 h-3 mr-1.5' />
          <div className='text-violet-100'>
            Share
          </div>
        </button>
      </div>
      {error && <ErrorMessage message={error} />}
      {grants?.length > 0 &&
        <div className='py-2 max-h-40 overflow-y-auto'>
          {grants?.filter(grant => grant.resource !== 'infra').map(item => (
            <div className='flex justify-between items-center' key={item.id}>
              <p className='text-2xs'>{item.resource}</p>
              <div>
                <select
                  id='role'
                  name='role'
                  className='w-full pl-3 pr-1 py-2 border-gray-300 focus:outline-none text-2xs text-gray-400 bg-transparent'
                  defaultValue={item.privilege}
                  onChange={e => handleUpdateGrant(e.target.value, item.resource, item.id, item.user)}
                >
                  {options.map((option) => (
                    <option key={option} value={option}>{option}</option>
                  ))}
                </select>
              </div>
            </div>
          ))}
        </div>}
      {grants?.filter(grant => grant.resource !== 'infra').length === 0 &&
        <div className='text-2xs text-gray-400 italic w-2/3'>
          No access
        </div>}
    </>
  )
}
