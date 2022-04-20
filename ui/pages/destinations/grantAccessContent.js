import useSWR, { useSWRConfig } from 'swr'
import { useState } from 'react'
import styled from 'styled-components'

import InputDropdown from '../../components/inputDropdown'

const GrantNewContainer = styled.div`
  display: grid;
  align-items: center;
  grid-template-columns: 80% auto;
  gap: .5rem;
  box-sizing: border-box;
  padding: 0 2rem 1.75rem 0;
`

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
  const { data: user } = useSWR(`/v1/identities/${id}`, { fallbackData: { name: '' } })

  return (
    <p>{user.name}</p>
  )
}

export default ({ id }) => {
  const options = ['view', 'edit', 'admin', 'remove']

  const { data: destination } = useSWR(`/v1/destinations/${id}`)
  const { data: list } = useSWR(() => `/v1/grants?resource=${destination.name}`)
  const { mutate } = useSWRConfig()

  const [grantNewEmail, setGrantNewEmail] = useState('')
  const [role, setRole] = useState('view')

  const grantPrivilege = (id, privilege = role) => {
    fetch('/v1/grants', {
      method: 'POST',
      body: JSON.stringify({ subject: id, resource: destination.name, privilege })
    })
      .then((response) => response.json())
      .then((data) => {
        console.log('data:', data)
        mutate(`/v1/grants?resource=${destination.name}`)
      })
      .finally(() => {
        setGrantNewEmail('')
      // setRole('view')
      })
  }

  const handleKeyDownEvent = (key) => {
    if (key === 'Enter' && grantNewEmail.length > 0) {
      handleShareGrant()
    }
  }

  const handleShareGrant = () => {
    fetch(`/v1/identities?name=${grantNewEmail}`)
      .then((response) => response.json())
      .then((data) => {
        if (data.length === 0) {
          fetch('/v1/identities', {
            method: 'POST',
            body: JSON.stringify({ name: grantNewEmail, kind: 'user' })
          })
            .then((response) => response.json())
            .then((user) => {
              grantPrivilege(user.id)
            })
            .finally(() => {
              setGrantNewEmail('')
              // setRole('view')
            })
        } else {
          grantPrivilege(data[0].id)
        }
      })
      .catch((error) => {
        console.log(error)
      })
  }

  const handleUpdateGrant = (privilege, grantId, userId) => {
    console.log({privilege, grantId, userId})
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
      <div>
        <GrantNewContainer>
          <InputDropdown
            type='email'
            value={grantNewEmail}
            placeholder='email'
            optionType='role'
            options={options.filter((item) => item !== 'remove')}
            handleInputChange={e => setGrantNewEmail(e.target.value)}
            handleSelectOption={e => setRole(e.target.value)}
            handleKeyDown={(e) => handleKeyDownEvent(e.key)}
          />
          <button
            onClick={() => handleShareGrant()}
            disabled={grantNewEmail.length === 0}
            type='button'
            className='bg-gradient-to-tr from-indigo-300 to-pink-100 hover:from-indigo-200 hover:to-pink-50 p-0.5 my-2 mx-auto'
          >
            <div className='bg-black flex items-center text-sm px-6 py-2'>
              Share
            </div>
          </button>
        </GrantNewContainer>
      </div>
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
