import useSWR, { useSWRConfig } from 'swr'
import { useState } from 'react'

import { validateEmail } from '../lib/email'

import InputDropdown from '../components/input'
import ErrorMessage from '../components/error-message'
import InfoModal from './modals/info'

function Grant ({ id }) {
  if (!id) {
    return null
  }

  const { data: user } = useSWR(`/v1/identities/${id.replace('i:', '')}`, { fallbackData: { name: '' } })

  return (
    <p>{user.name}</p>
  )
}

export default function ({ id, modalOpen, handleCloseModal }) {
  const { data: destination } = useSWR(`/v1/destinations/${id}`)
  const { data: list } = useSWR(() => `/v1/grants?resource=${destination.name}`)
  const { mutate } = useSWRConfig()

  const [email, setEmail] = useState('')
  const [error, setError] = useState('')
  const [grantError, setGrantError] = useState('')
  const [role, setRole] = useState('view')

  const options = ['view', 'edit', 'admin', 'remove']

  const grantPrivilege = async (id, privilege = role, exist = false, deleteGrantId) => {
    const newGrant = { ...list.filter(item => item?.subject === id)[0], privilege }

    mutate(`/v1/grants?resource=${destination.name}`, async grants => {
      const res = await fetch('/v1/grants', {
        method: 'POST',
        body: JSON.stringify({ subject: id, resource: destination.name, privilege })
      })
      
      const data = await res.json()

      if(exist) {
        await fetch(`/v1/grants/${deleteGrantId}`, { method: 'DELETE' })
      }

      setEmail('')

      return [...(grants || []).filter(grant => grant?.subject !== id), data]
    }, { optimisticData: [...list.filter(item => item?.subject !== id), newGrant]})
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

        if(!res.ok) {
          throw data
        }

        if (data.length === 0) {
          res = await fetch('/v1/identities', {
                  method: 'POST',
                  body: JSON.stringify({ name: email, kind: 'user' })
                })
          const user = await res.json()

          await grantPrivilege('i:' + user.id)
          setEmail('')
          setRole('view')
        } else {
          grantPrivilege('i:' + data[0].id)
        }
      } catch(e) {
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

      return grants.filter(item => item?.id !== grantId)
    }, { optimisticData: list.filter(item => item?.id !== grantId) })
  }

  return (
    <InfoModal
      header='Grant'
      handleCloseModal={handleCloseModal}
      modalOpen={modalOpen}
      iconPath='/grant-access-color.svg'
    >
      <div className={`flex gap-1 mt-3 ${error ? 'mb-2' : 'mb-8'}`}>
        <div className='flex-2 w-full'>
          <InputDropdown
            type='email'
            value={email}
            placeholder='email'
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
          className='bg-gradient-to-tr from-indigo-300 to-pink-100 rounded-full hover:from-indigo-200 hover:to-pink-50 p-0.5 mx-auto disabled:opacity-30'
        >
          <div className='bg-black flex items-center text-sm rounded-full px-12 py-3'>
            Share
          </div>
        </button>
      </div>
      {error && <ErrorMessage message={error} />}

      {list && list.length > 0 &&
        <section className='py-2'>
          {list.map((item) => (
            <div className='flex justify-between items-center px-4' key={item.id + item.subject}>
              <Grant id={item.subject} />
              <div>
                <select
                  id='role'
                  name='role'
                  className='w-full pl-3 pr-1 py-2 border-gray-300 focus:outline-none sm:text-sm bg-transparent'
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
        </section>}
        {grantError && <ErrorMessage message={grantError} />}

    </InfoModal>
  )
}
