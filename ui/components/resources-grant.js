import { useState } from "react";
import useSWR, { mutate } from "swr";
import { PlusIcon } from "@heroicons/react/outline";

import InputDropdown from '../components/input'
import ErrorMessage from '../components/error-message'

export default function ({ id }) {
  const { data: grants } = useSWR(`/v1/identities/${id}/grants`)

  const [infrastructure, setInfrastructure] = useState('')
  const [error, setError] = useState('')
  const [role, setRole] = useState('view')

  const options = ['view', 'edit', 'admin', 'remove']

  const handleInputChang = value => {
    setInfrastructure(value)
    setError('')
  }

  const handleKeyDownEvent = key => {
    if (key === 'Enter' && email.length > 0) {
      handleShareGrant()
    }
  }

  const handleShareGrant = () => {
    console.log('handleShareGrant')
  }

  const updateGrant = (privilege, resource, user, deleteGrantId) => {
    mutate(`/v1/grants?resource=${resource}`, async grantsList => {
      const res = await fetch('/v1/grants', {
        method: 'POST',
        body: JSON.stringify({ subject: 'i:' + user, resource, privilege })
      })

      const data = await res.json()

      await fetch(`/v1/grants/${deleteGrantId}`, { method: 'DELETE' })

      return [...(grantsList || []).filter(grant => grant?.subject !== 'i:' + user), data]
    })
  }

  const handleUpdateGrant = (privilege, resource, grantId, user) => {
    console.log('updating grant')
    if (privilege !== 'remove') {
      console.log('updating', resource)
      return updateGrant(privilege, resource, user, grantId)
    }

    mutate(`/v1/identities/${user}/grants`, async userGrantsList => {
      await fetch(`/v1/grants/${grantId}`, { method: 'DELETE'})

      return userGrantsList?.filter(item => item?.id !== grantId)
    }, { optimisticData: grants?.filter(item => item?.id !== grantId) })
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
          className='bg-gradient-to-tr disabled:opacity-30 disabled:transform-none disabled:transition-none cursor-pointer disabled:cursor-default from-indigo-300 to-pink-100 hover:from-indigo-200 hover:to-pink-50 p-0.5 mt-4 mr-auto sm:ml-4 sm:mt-0 rounded-md'
        >
          <div className='bg-black flex items-center text-xs rounded-md hover:text-pink-50 px-3 py-3'>
            <PlusIcon className='w-3 h-3 mr-1.5' />
            <div className='text-pink-100'>
              Share
            </div>
          </div>
        </button>
      </div>
      {error && <ErrorMessage message={error} />}
      {grants?.length > 0 &&
        <div className='py-2 max-h-40 overflow-y-auto'>
          {grants?.filter((grant) => grant.resource !== 'infra').map((item) => (
            <div className='flex justify-between items-center' key={item.id}>
              <p className='text-paragraph'>{item.resource}</p>
              <div>
                <select
                  id='role'
                  name='role'
                  className='w-full pl-3 pr-1 py-2 border-gray-300 focus:outline-none text-paragraph text-gray-400 bg-transparent'
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
      {grants?.filter((grant) => grant.resource !== 'infra').length === 0 && <div className='text-paragraph text-gray-400 italic w-2/3'>
        *TODO - this user doesn't have any access
      </div>}
    </>
  )
}