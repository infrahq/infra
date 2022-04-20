import styled from 'styled-components'
import { useState } from 'react'
import useSWR, { useSWRConfig } from 'swr'

import InputDropdown from '../../components/inputDropdown'

const AddAdminContainer = styled.div`
  display: grid;
  align-items: center;
  grid-template-columns: 75% auto;
  gap: .5rem;
  box-sizing: border-box;
  padding: 0 0 1rem 0;
  width: 70%;
`

const AdminItem = styled.div`
  display: grid;
  align-items: flex-start;
  grid-template-columns: 1fr min-content;
  box-sizing: border-box;
  width: 50%;
`

const AdminList = styled.div`
  & > *:first-child {
    padding-top: 1.5rem;
  }

  & > *:not(:first-child) {
    padding-top: 1rem;
  }
`

const AdminName = ({ id }) => {
  const { data: user } = useSWR(`/v1/identities/${id}`, { fallbackData: { name: '' } })

  return (
    <p>{user.name}</p>
  )
}

export default function () {
  const { mutate } = useSWRConfig()
  const { data: adminList } = useSWR(() => '/v1/grants?resource=infra', { fallbackData: [] })

  const [adminEmail, setAdminEmail] = useState('')

  const grantAdminAccess = (id) => {
    fetch('/v1/grants', {
      method: 'POST',
      body: JSON.stringify({ subject: id, resource: 'infra', privilege: 'admin' })
    })
      .then(() => {
        mutate('/v1/grants?resource=infra')
        setAdminEmail('')
      }).catch((error) => {
        console.log(error)
      })
  }

  const handleKeyDownEvent = (key) => {
    if (key === 'Enter' && adminEmail.length > 0) {
      handleAddAdmin()
    }
  }

  const handleAddAdmin = () => {
    fetch(`/v1/identities?name=${adminEmail}`)
      .then((response) => response.json())
      .then((data) => {
        if (data.length === 0) {
          fetch('/v1/identities', {
            method: 'POST',
            body: JSON.stringify({ name: adminEmail, kind: 'user' })
          })
            .then((response) => response.json())
            .then((user) => {
              grantAdminAccess(user.id)
            })
            .catch((error) => {
              console.log(error)
            })
        } else {
          grantAdminAccess(data[0].id)
        }
      })
  }

  const handleDeleteAdmin = (id) => {
    fetch(`/v1/grants/${id}`, { method: 'DELETE' })
      .then(() => {
        mutate('/v1/grants?resource=infra')
      })
      .catch((error) => {
        console.log(error)
      })
  }

  return (
    <>
      <h3 className='text-lg font-bold mb-4'>Admins</h3>
      <AddAdminContainer>
        <InputDropdown
          type='email'
          value={adminEmail}
          placeholder='email'
          hasDropdownSelection={false}
          handleInputChange={e => setAdminEmail(e.target.value)}
          handleKeyDown={(e) => handleKeyDownEvent(e.key)}
        />
        <button
          onClick={() => handleAddAdmin()}
          disabled={adminEmail.length === 0}
          type='button'
          className='bg-gradient-to-tr from-indigo-300 to-pink-100 hover:from-indigo-200 hover:to-pink-50 p-0.5 my-2 mx-auto'
        >
          <div className='bg-black flex items-center text-sm px-12 py-2'>
            Add
          </div>
        </button>
      </AddAdminContainer>
      <p className='text-gray-400'>These  users have full administration privileges</p>
      {adminList && adminList.length > 0 &&
        <AdminList>
          {adminList.map((admin) => (
            <div key={admin.id}>
              <AdminItem>
                <AdminName id={admin.subject} />
                <div className='cursor-pointer' onClick={() => handleDeleteAdmin(admin.id)}>&#10005;</div>
              </AdminItem>
            </div>
          ))}
        </AdminList>}
    </>
  )
}
