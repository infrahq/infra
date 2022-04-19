import useSWR, { useSWRConfig } from 'swr'
import Head from "next/head"
import { useState } from "react"
import styled from "styled-components"

import InputDropdown from './inputDropdown'
import GrantSelectionDropdown from './grantSelectionDropdown'

const GrantNewContainer = styled.div`
	display: grid;
  align-items: center;
  grid-template-columns: 85% auto;
  gap: 5px;
  box-sizing: border-box;
`

const GrantList = styled.section`
  & > *:first-child {
    padding-top: 2rem;
  }

  & > *:not(:first-child) {
    padding-top: 1rem;
  }
`

const GrantListItem = styled.div`
  display: flex;
  flex-direction: row;
  justify-content: space-between;
  align-items: center;
`

const Grant = ({ id }) => {
  const { data: user } = useSWR(`/v1/identities/${id}`, {fallbackData: {name: ''}})

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
      setRole('view')
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
          setRole('view')
        })
      } else {
        grantPrivilege(data[0].id)
      }
    })
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
              type="button"
              className='mt-3 w-full inline-flex justify-center rounded-md border border-gray-300 shadow-sm px-4 py-2 bg-black text-white font-medium hover:text-gray-300 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-indigo-500 sm:mt-0 sm:w-auto sm:text-sm'
            >
              Share
            </button>
        </GrantNewContainer>
      </div>
      {list && list.length > 0 && <GrantList>
        {
          list.map((item) => (
            <GrantListItem key={item.id}>
              <Grant id={item.subject} />
              <div>
                <GrantSelectionDropdown
                  optionType='role'
                  options={options}
                  selectedValue={item.privilege}
                  handleChangeSelection={e => handleUpdateGrant(e.target.value, item.id, item.subject)}
                />
              </div>
            </GrantListItem>
          ))
        }
      </GrantList>}
		</>
	)
}
