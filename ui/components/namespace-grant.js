import useSWR, { useSWRConfig } from 'swr'
import { useState } from 'react'
import { PlusIcon } from '@heroicons/react/outline'

import InputDropdown from '../components/input'
import ErrorMessage from '../components/error-message'

function User ({ id }) {
  if (!id) {
    return null
  }

  const { data: user } = useSWR(`/api/users/${id}`, { fallbackData: { name: '' } })

  return (
    <p className='text-2xs'>{user.name}</p>
  )
}

export default function ({ destinationId, namespaceName }) {
  const { data: destination } = useSWR(`/api/destinations/${destinationId}`)
  const { data: { items } = {} } = useSWR(() => `/api/grants?resource=${destination.name}`)
  const { data: { items: namespaceGrants } = {}} = useSWR(() => `/api/grants?resource=${namespaceName}`)

  console.log(namespaceGrants)
  
  const InherittedAccessList = items?.filter(item => !!item.user)

  const [name, setName] = useState('')
  const [error, setError] = useState('')
  const [grantError, setGrantError] = useState('')
  const [role, setRole] = useState('view')

  const options = ['view', 'edit', 'admin', 'remove']

  const handleInputChange = value => {
    // setName(value)
    // setError('')
  }

  const handleKeyDownEvent = key => {
    // if (key === 'Enter' && name.length > 0) {
    //   handleShareGrant()
    // }
  }

  const handleShareGrant = async () => {
    // setError('')
    // try {
    //   const res = await fetch(`/api/users?name=${name}`)
    //   const data = await res.json()

    //   if (!res.ok) {
    //     throw data
    //   }

    //   if (data?.items?.length === 0) {
    //     setError('User does not exist')
    //   } else {
    //     grantPrivilege(data?.items?.[0]?.id)
    //   }
    // } catch (e) {
    //   setGrantError(e.message || 'something went wrong, please try again later.')
    // }
  }

  return (
    <>
      <div className={`flex gap-1 mt-3 ${error ? 'mb-2' : 'mb-4'}`}>
        <div className='flex-1'>
          <InputDropdown
            value={name}
            placeholder='Username'
            error={error}
            optionType='role'
            options={options.filter((item) => item !== 'remove')}
            handleInputChange={e => handleInputChange(e.target.value)}
            handleSelectOption={e => setRole(e.target.value)}
            handleKeyDown={(e) => handleKeyDownEvent(e.key)}
            selectedItem={role}
          />
        </div>
        <button
          onClick={() => handleShareGrant()}
          disabled={name.length === 0}
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
      {InherittedAccessList?.length > 0 && 
        <div className='py-2 overflow-y-auto'>
        {InherittedAccessList?.sort((a, b) => (a.user).localeCompare(b.user)).map(item => (
          <div className='grid grid-cols-5 gap-1' key={item.id}>
          <div className='col-span-3 py-2'><User id={item.user} /></div>
          <div className='my-2 text-2xs text-gray-400 border rounded-sm px-2 mx-auto bg-gray-800 border-gray-800'>
            inherited
          </div>
          <div className='py-2 text-2xs text-gray-400'>
            {item.privilege}
          </div>
        </div>
        ))}
      </div>
      }
      {namespaceGrants?.length > 0 &&
        <div className='py-2 max-h-40 overflow-y-auto'>
          {namespaceGrants?.sort((a, b) => (a.user).localeCompare(b.user)).map(item => (
            <div className='flex justify-between items-center' key={item.id}>
              <User id={item.user} />
              <div>
                <select
                  id='role'
                  name='role'
                  className='w-full pl-3 pr-1 py-2 border-gray-300 focus:outline-none text-2xs text-gray-400 bg-transparent'
                  defaultValue={item.privilege}
                  onChange={e => handleUpdateGrant(e.target.value, item.id, item.user)}
                >
                  {options.map((option) => (
                    <option key={option} value={option}>{option}</option>
                  ))}
                </select>
              </div>
            </div>
          ))}
        </div>}
      {InherittedAccessList?.length === 0 && namespaceGrants?.length === 0 && (
        <div className='text-2xs text-gray-400 italic w-2/3'>
          Share access to this cluster by inviting your team and assigning their roles.
        </div>
      )}
      {grantError && <ErrorMessage message={grantError} />}
    </>
  )
}