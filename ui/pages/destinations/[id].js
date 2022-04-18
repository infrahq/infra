import useSWR, { useSWRConfig } from 'swr'
import useSWRImmutable from 'swr/immutable'
import Head from "next/head"
import { useRouter } from "next/router"
import { useState } from "react"
import styled from "styled-components"

import ExitButton from "../../components/ExitButton"
import InputDropdown from '../../components/InputDropdown'
import GrantSelectionDropdown from '../../components/GrantSelectionDropdown'

const DetailsContainer = styled.div`
	position: relative;
`

const ContainerContent = styled.section`
  margin-left: auto;
  margin-right: auto;
  max-width: 40rem;
  padding-top: 1.5rem;

  & > *:not(:first-child) {
    padding-top: 1.75rem;
  }
`

const Nav = styled.section`
  position: absolute;
  right: .5rem;
  top: .5rem;
`

const GrantNewContainer = styled.div`
	display: grid;
  align-items: center;
  grid-template-columns: 1fr min-content;
  box-sizing: border-box;
`

const GrantList = styled.section`
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

const DestinationDetails = () => {
  const router = useRouter()
	const { id } = router.query
	
	const { data: destination } = useSWR(`/v1/destinations/${id}`)
	const { data: list } = useSWR(() => `/v1/grants?resource=${destination.name}`)
  const { mutate } = useSWRConfig()

	const [grantNewEmail, setGrantNewEmail] = useState('')
  const [role, setRole] = useState('view')

  const options = ['view', 'edit', 'admin']

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
  }

	const handleShareGrant = () => {
		console.log('email:', grantNewEmail, 'role:', role)
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
      grantPrivilege(userId, privilege)
    })
  }

	return (
		<>
			<Head>
				<title>Infra - Destination Details</title>
			</Head>
			<DetailsContainer>
				<ContainerContent>
					<h2>Grant</h2>
					<div>
						<GrantNewContainer>
                <InputDropdown
                  type='email'
                  value={grantNewEmail}
                  placeholder='email'
                  optionType='role'
                  options={options}
                  handleInputChange={e => setGrantNewEmail(e.target.value)}
                  handleSelectOption={e => setRole(e.target.value)}
                />
                <div className='rounded overflow-hidden bg-gradient-to-tr from-cyan-100 to-pink-300 ml-2'>
                  <button
                    onClick={() => handleShareGrant()}
                    disabled={grantNewEmail.length === 0}
                    type="button"
                    className="flex items-center m-px px-2 py-1 rounded bg-black hover:bg-gray-900 transition-all duration-200 disabled:opacity-90"
                  >
                    Share
                  </button>
                </div>
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
				</ContainerContent>
				<Nav>
					<ExitButton previousPage='/destinations' />
				</Nav>
			</DetailsContainer>
		</>
	)
}

export default DestinationDetails


