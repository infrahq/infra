import useSWR, { useSWRConfig } from 'swr'
import { useState } from 'react'

import { validateEmail } from '../lib/email'

import InputDropdown from '../components/input'
import ErrorMessage from '../components/error-message'
import { PlusIcon } from '@heroicons/react/outline'

function Grant ({ id }) {
  if (!id) {
    return null
  }

  const { data: user } = useSWR(`/v1/identities/${id.replace('i:', '')}`, { fallbackData: { name: '' } })

  return (
    <p className='text-xs'>{user.name}</p>
  )
}

export default function ({ id }) {
  const { data: destination } = useSWR(`/v1/destinations/${id}`)
  const { data: list } = useSWR(() => `/v1/grants?resource=${destination.name}`)
  const { mutate } = useSWRConfig()

  const [email, setEmail] = useState('')
  const [error, setError] = useState('')
  const [grantError, setGrantError] = useState('')
  const [role, setRole] = useState('view')

  const options = ['view', 'edit', 'admin', 'remove']

  const grantPrivilege = async (id, privilege = role, exist = false, deleteGrantId) => {
    mutate(`/v1/grants?resource=${destination.name}`, async grants => {
      const res = await fetch('/v1/grants', {
        method: 'POST',
        body: JSON.stringify({ subject: id, resource: destination.name, privilege })
      })

      const data = await res.json()

      if (exist) {
        await fetch(`/v1/grants/${deleteGrantId}`, { method: 'DELETE' })
      }

      setEmail('')

      return [...(grants || []).filter(grant => grant?.subject !== id), data]
    })
  }

  const handleInputChang = value => {
    setEmail(value)
    setError('')
  }

  const handleKeyDownEvent = key => {
    if (key === 'Enter' && email.length > 0) {
      handleShareGrant()
    }
  }

  const handleShareGrant = async () => {
    if (validateEmail(email)) {
      setError('')
      try {
        let res = await fetch(`/v1/identities?name=${email}`)
        const data = await res.json()

        if (!res.ok) {
          throw data
        }

        if (data.length === 0) {
          res = await fetch('/v1/identities', {
            method: 'POST',
            body: JSON.stringify({ name: email })
          })
          const user = await res.json()

          await grantPrivilege('i:' + user.id)
          setEmail('')
          setRole('view')
        } else {
          grantPrivilege('i:' + data[0].id)
        }
      } catch (e) {
        setGrantError(e.message || 'something went wrong, please try again later.')
      }
    } else {
      setError('Invalid email')
    }
  }

  const handleUpdateGrant = (privilege, grantId, userId) => {
    if (privilege !== 'remove') {
      return grantPrivilege(userId, privilege, true, grantId)
    }

    mutate(`/v1/grants?resource=${destination.name}`, async grants => {
      await fetch(`/v1/grants/${grantId}`, { method: 'DELETE' })

      return grants?.filter(item => item?.id !== grantId)
    }, { optimisticData: list?.filter(item => item?.id !== grantId) })
  }

  return (
    <>
      <div className={`flex gap-1 mt-3 ${error ? 'mb-2' : 'mb-4'}`}>
        <div className='flex-1'>
          <InputDropdown
            type='email'
            value={email}
            placeholder='Email'
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
          disabled={email.length === 0}
          type='button'
          className='flex items-center border border-violet-300 disabled:opacity-30 disabled:transform-none disabled:transition-none cursor-pointer disabled:cursor-default mt-4 mr-auto sm:ml-4 sm:mt-0 rounded-md text-xs px-3 py-3'
        >
          <PlusIcon className='w-3 h-3 mr-1.5' />
          <div className='text-violet-100'>
            Share
          </div>
        </button>
      </div>
      {error && <ErrorMessage message={error} />}
      {list?.length > 0 &&
        <div className='py-2 max-h-40 overflow-y-auto'>
          {list?.sort((a, b) => (a.subject).localeCompare(b.subject)).map((item) => (
            <div className='flex justify-between items-center' key={item.id}>
              <Grant id={item.subject} />
              <div>
                <select
                  id='role'
                  name='role'
                  className='w-full pl-3 pr-1 py-2 border-gray-300 focus:outline-none text-xs text-gray-400 bg-transparent'
                  defaultValue={item.privilege}
                  onChange={e => handleUpdateGrant(e.target.value, item.id, item.subject)}
                >
                  {options.map((option) => (
                    <option key={option} value={option}>{option}</option>
                  ))}
                </select>
              </div>
            </div>
          ))}
        </div>}
      {list?.length === 0 && (
        <div className='text-xs text-gray-400 italic w-2/3'>
          *Share access to this cluster by inviting your team and assigning their roles.
        </div>
      )}
      {grantError && <ErrorMessage message={grantError} />}
    </>
  )
}
