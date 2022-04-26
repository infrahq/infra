import useSWR, { useSWRConfig } from 'swr'
import { useState } from 'react'

import { validateEmail } from '../lib/email'

import InputDropdown from '../components/input'
import ErrorMessage from '../components/error-message'
import InfoModal from './modals/info'

function Grant ({ id }) {
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
  const [role, setRole] = useState('view')

  const options = ['view', 'edit', 'admin', 'remove']

  const grantPrivilege = async (id, privilege = role) => {
    // TODO: THIS IS CREATING EXTRA ENTRY EVERYTIME UPDATES
    mutate(`/v1/grants?resource=${destination.name}`, async grants => {
      const res = await fetch('/v1/grants', {
        method: 'POST',
        body: JSON.stringify({ subject: id, resource: destination.name, privilege })
      })
      const data = await res.json()

      setEmail('')

      console.log('id:', id)
      console.log('grants:', grants)
      console.log('data:', data)
      console.log('filter:', (grants || []).filter(grant => grant?.subject !== id))
      console.log('return data:', [...(grants || []).filter(grant => grant?.subject !== id), data])

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
      await fetch(`/v1/identities?name=${email}`)
        .then((response) => response.json())
        .then(async (data) => {
          if (data.length === 0) {
            await fetch('/v1/identities', {
              method: 'POST',
              body: JSON.stringify({ name: email, kind: 'user' })
            })
              .then((response) => response.json())
              .then((user) => grantPrivilege('i:' + user.id))
              .finally(() => setEmail(''))
          } else {
            grantPrivilege(data[0].id)
          }
        })
        .catch((error) => console.error(error))
    } else {
      setError('Invalid email')
    }
  }

  const handleUpdateGrant = (privilege, grantId, userId) => {
    if (privilege !== 'remove') {
      return grantPrivilege(userId, privilege)
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
            optionType='role'
            options={options.filter((item) => item !== 'remove')}
            handleInputChange={e => handleInputChang(e.target.value)}
            handleSelectOption={e => setRole(e.target.value)}
            handleKeyDown={(e) => handleKeyDownEvent(e.key)}
            error={error}
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
            <div className='flex justify-between items-center px-4' key={item.id}>
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

    </InfoModal>
  )
}
