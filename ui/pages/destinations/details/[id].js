import useSWR from 'swr'
import useSWRImmutable from 'swr/immutable'
import Head from "next/head"
import { useRouter } from "next/router"
import { useEffect, useState } from "react"
import styled from "styled-components"
import { getDestinationsList } from ".."

import ExitButton from "../../../components/ExitButton"

const DetailsContainer = styled.div`
	position: relative;
`

const ContainerContent = styled.section`
  margin-left: auto;
  margin-right: auto;
  max-width: 24rem;
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

const GrantNewShareButton = styled.button`

`

const GrantContainer = styled.div``

const GrantNewEmailContainer = styled.div`
	background-color: transparent;
	display: flex;
	align-items: center;
	border: 1px solid #e5e5e5;
	border-radius: 3px;
	padding: 2.5px;
	justify-content: space-between;
	margin-right: 8px;
`

const GrantNewContainerInput = styled.div`
	max-width: 80%;
	display: flex;
  flex-wrap: wrap;
  align-content: flex-start;
  background-color: #ffffff;
  border-radius: 3px;
  flex-grow: 1;
`

const InputContainer = styled.input`
	background: #000000;
  border: 1px solid transparent;
  flex-grow: 1;
  width: 90px;
  color: #ffffff;
  cursor: default;
`

const GrantNewContainerDropdown = styled.div``

const GrantListContainer = styled.div``

export const getGrantsList = (name) => {
	console.log(name)
  const getGrantsListURL = `/v1/grants?resource=${name}`
  const getGrants = url => fetch(url).then(response => response.json())
  const { data, error } = useSWRImmutable(getGrantsListURL, getGrants)
  
  return {
    list: data,
    isLoading: !error && !data,
    isError: error
  }
}

const DestinationDetails = () => {
  const router = useRouter()
	const { id } = router.query
	
	const { destinations } = getDestinationsList()
	const { list } = getGrantsList(destinations && destinations.find((destination) => destination.id === id).name)

	const [destination, setDestination] = useState(null)
	const [name, setName] = useState(null)
	const [clusterType, setClusterType] = useState(null)
	
	const [grantNewEmail, setGrantNewEmail] = useState('')

	useEffect(() => {
		const currentDestination = destinations && destinations.find((destination) => destination.id === id)

		setDestination(currentDestination)
		getTitle(currentDestination)
	}, [])

	const getTitle = (destination) => {
		const destinationName = destination && destination.name
		if (destinationName) {
			setClusterType(destinationName.substr(0, destinationName.indexOf('.')) + ' cluster')
			setName(destinationName.substr( destinationName.indexOf('.') + 1 ))
		}
	}

	const handleShareGrant = () => {
		console.log('email:', grantNewEmail)
	}

	return (
		<>
			<Head>
				<title>Infra - Destination Details</title>
			</Head>
			<DetailsContainer>
				<ContainerContent>
					{destination && <h1>{name}</h1>}
					{destination && <p>{clusterType}</p>}
					<GrantContainer>
						<h2>Grant</h2>
						<GrantNewContainer>
							<GrantNewEmailContainer>
								<GrantNewContainerInput>
									<InputContainer 
										type={'email'} 
										value={grantNewEmail} 
										onChange={e => setGrantNewEmail(e.target.value)}
									/>
								</GrantNewContainerInput>
								<GrantNewContainerDropdown></GrantNewContainerDropdown>
							</GrantNewEmailContainer>
							<GrantNewShareButton onClick={() => handleShareGrant()}>Share</GrantNewShareButton>
						</GrantNewContainer>
						{list && <GrantListContainer>
							<p>list list list</p>
						</GrantListContainer>}
					</GrantContainer>
				</ContainerContent>
				<Nav>
					<ExitButton previousPage='/destinations' />
				</Nav>
			</DetailsContainer>
		</>
	)
}

export default DestinationDetails
