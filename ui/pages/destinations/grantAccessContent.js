import useSWR, { useSWRConfig } from 'swr'
import { useState } from 'react'
import styled from 'styled-components'

import { validateEmail } from '../../lib/email'

import InputDropdown from '../../components/input-dropdown'
import ErrorMessage from '../../components/error-message'

const GrantList = styled.section`
  max-height: 20rem;
  overflow: auto;
  width: 95%;
  padding: 0 1.25rem;
`

const GrantListItem = styled.div`
  display: flex;
  flex-direction: row;
  justify-content: space-between;
  align-items: center;
`

const Grant = ({ id }) => {
  const { data: user } = useSWR(`/v1/identities/${id.replace('i:', '')}`, { fallbackData: { name: '' } })

  return (
    <p>{user.name}</p>
  )
}

export default ({ id }) => {
  const { data: destination } = useSWR(`/v1/destinations/${id}`)
  const { data: list } = useSWR(() => `/v1/grants?resource=${destination.name}`)
  const { mutate } = useSWRConfig()

  const [email, setEmail] = useState('')
  const [error, setError] = useState('')
  const [role, setRole] = useState('view')

  const options = ['view', 'edit', 'admin', 'remove']

  const grantPrivilege = (id, privilege = role) => {
    fetch('/v1/grants', {
      method: 'POST',
      body: JSON.stringify({ subject: id, resource: destination.name, privilege })
    })
      .then((response) => response.json())
      .then(() => mutate(`/v1/grants?resource=${destination.name}`))
      .finally(() => setEmail(''))
  }

  const handleInputChang = (value) => {
    setEmail(value)
    setError('')
  }

  const handleKeyDownEvent = (key) => {
    if (key === 'Enter' && email.length > 0) {
      handleShareGrant()
    }
  }

  const handleShareGrant = () => {
    if (validateEmail(email)) {
      setError('')
      fetch(`/v1/identities?name=${email}`)
        .then((response) => response.json())
        .then((data) => {
          if (data.length === 0) {
            fetch('/v1/identities', {
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
    fetch(`/v1/grants/${grantId}`, { method: 'DELETE' })
      .then(() => {
        if (privilege === 'remove') {
          mutate(`/v1/grants?resource=${destination.name}`)
        } else {
          grantPrivilege(userId, privilege)
        }
      })
  }

  return (
    <>
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
        <GrantList>
          {list.map((item) => (
            <GrantListItem key={item.id}>
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
            </GrantListItem>
          ))}
        </GrantList>}
    </>
  )
}
