import styled from "styled-components";
import { useState } from "react";
import useSWR, { useSWRConfig } from "swr";

import Dashboard from "../../components/dashboard";
import InputDropdown from "../../components/InputDropdown";


const AddAdminContainer = styled.div`
	display: grid;
  align-items: center;
  grid-template-columns: 1fr min-content;
  box-sizing: border-box;

  max-width: 40rem;
`

const AdminItem = styled.div`
  display: grid;
  align-items: center;
  grid-template-columns: 1fr min-content;
  box-sizing: border-box;

  max-width: 38rem;
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
    }).catch((error) => {
      console.log(error)
    })
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
        />
        <div className='rounded overflow-hidden bg-gradient-to-tr from-cyan-100 to-pink-300 ml-2'>
          <button
          onClick={() => handleAddAdmin()}
          disabled={adminEmail.length === 0}
          type="button"
          className="flex items-center m-px px-2 py-1 rounded bg-black hover:bg-gray-900 transition-all duration-200 disabled:opacity-90"
          >
          Add
          </button>
        </div>
      </AddAdminContainer>
      {adminList && adminList.length > 0 && <div>
        {adminList.map((admin) => (
          <>
            <AdminItem key={admin.id}>
              <AdminName id={admin.subject} />
              <div onClick={() => handleDeleteAdmin(admin.id)}>&#10005;</div>
            </AdminItem>
          </>
        ))}
        </div>}
    </Dashboard>
  )
}