import Head from "next/head"
import { useContext, useState } from "react"
import styled from 'styled-components'

import ActionButton from "../../../components/ActionButton"
import ExitButton from "../../../components/ExitButton"
import Header from "../../../components/Header"
import Input from "../../../components/Input"
import WarningContainer from "../../../components/WarningContainer"
import DestinationsContext, { DestinationsContextProvider } from "../../../store/DestinationsContext"

const SetupDestinationContainer = styled.section`
  position: relative;
`

const NavButton = styled.div`
  position: absolute;
  top: .5rem;
  right: .5rem;
`

const SetupDestinationContent = styled.div`
  margin-left: auto;
  margin-right: auto;
  max-width: 24rem;
  padding-top: 1.5rem;

  & > *:not(:first-child) {
    padding-top: 1.75rem;
  }
`

const Setup = () => {
  const { destinations } = useContext(DestinationsContext)
  const [name, setName] = useState('')
  const [isDuplicated, setIsDuplicated] = useState(false)

  const handleSetup = () => {
    const type = 'kubernetes'
    const destinationName = type + '.' + name
    setIsDuplicated(!isUnique(destinationName))

    console.log(destinationName)
  }

  const isUnique = (currentDestinationName) => {
    return destinations.filter((item) => item.name === currentDestinationName).length === 0
  }

  return (
    <DestinationsContextProvider>
      <Head>
        <title>Infra - Destinations</title>
      </Head>
      <SetupDestinationContainer>
        <SetupDestinationContent>
          <Header 
            header='Connect Your Kubernetes Cluster'
            subheader='Run the following command to connect your cluster'
          />
          <div>
            <Input 
              label='Provide a name for your cluster'
              value={name}
              onChange={e => setName(e.target.value)}
            />
          </div>
          { isDuplicated && 
            <WarningContainer text='the clouster already exists, please provide a different name' />
          }
          <ActionButton onClick={handleSetup} value='Next' />
        </SetupDestinationContent>      
        <NavButton>
          <ExitButton previousPage='/destinations' />
        </NavButton>
      </SetupDestinationContainer>
    </DestinationsContextProvider>
  )
}

export default Setup