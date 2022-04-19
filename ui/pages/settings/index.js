import styled from "styled-components";
import { useState } from "react";
import useSWR, { useSWRConfig } from "swr";

import Dashboard from "../../components/dashboard";
import InputDropdown from "../../components/inputDropdown";


const AddAdminContainer = styled.div`
	display: grid;
  align-items: center;
  grid-template-columns: 85% auto;
  gap: 5px;
  box-sizing: border-box;

  max-width: 30rem;
`

const AdminItem = styled.div`
  display: grid;
  align-items: center;
  grid-template-columns: 1fr min-content;
  box-sizing: border-box;

  max-width: 27rem;
`

const AdminList = styled.div`
  & > *:first-child {
    padding-top: 2rem;
  }

  & > *:not(:first-child) {
    padding-top: 1rem;
  }
`

const AdminName = ({ id }) => {
  const { data: user } = useSWR(`/v1/identities/${id}`, {fallbackData: {name: ''}})

  return (
    <p>{user.name}</p>
  )
}
  
export default function () {
  const { mutate } = useSWRConfig()
  const { data: adminList } = useSWR(() => '/v1/grants?resource=infra', {fallbackData: []})
  
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
    <Dashboard>
      <div>Admins</div>
      <p>These  users have full administration privileges</p>
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
          type="button"
          className='mt-3 w-full inline-flex justify-center rounded-md border border-gray-300 shadow-sm px-4 py-2 bg-black text-white font-medium hover:text-gray-300 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-indigo-500 sm:mt-0 sm:w-auto sm:text-sm'
        >
          Add
        </button>
      </AddAdminContainer>
      {adminList && adminList.length > 0 && <AdminList>
        {adminList.map((admin) => (
          <div key={admin.id}>
            <AdminItem>
              <AdminName id={admin.subject} />
              <div className='cursor-pointer' onClick={() => handleDeleteAdmin(admin.id)}>&#10005;</div>
            </AdminItem>
          </div>
        ))}
        </AdminList>}
    </Dashboard>
  )
}