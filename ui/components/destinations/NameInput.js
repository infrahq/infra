import { useContext, useState, useEffect } from "react"
import styled from 'styled-components'

import Input from "../Input"

import DestinationsContext from "../../store/DestinationsContext"
import axios from "axios"

const NameContainer = styled.div`
	display: flex;
	flex-direction: row;
	justify-content: space-between;
`

const InputContainer = styled.div`
	width: 78%;
`

const NextButton = styled.button`
  background-color: transparent;
  cursor: pointer;
  color: white;
  border: 1px solid rgba(255,255,255,0.25);
  box-sizing: border-box;
  border-radius: 1px;
  width: 20%;
`

const NameInput = () => {
  const { connected, destinations, accessKey, updateCurrentDestinationName, updateEnabledCommandInput, updateAccessKey, updateConnected } = useContext(DestinationsContext)
  const [name, setName] = useState('')
  const [numDestinations, setNumDestinations] = useState(0)

  useEffect(() => {
    const handleDestinationConnection = () => {
      if(accessKey && name.length > 0) {
        axios.get(`/v1/destinations?name=${name}`)
          .then((response) => {
            if (!connected) {
              if (response.data.length === numDestinations) {
                pollingTimeout = setTimeout(handleDestinationConnection, 5000)
              } else {
                updateConnected(true)
                clearTimeout(pollingTimeout)
              }
            } 
          })
          .catch((error) => {
            console.log(error)
            clearTimeout(pollingTimeout)
          })
      }
    }

    const pollingTimeout = setTimeout(handleDestinationConnection, 5000)

    return () => {
      clearTimeout(pollingTimeout)
    };
  }, [accessKey]);

  const handleNext = () => {
    const type = 'kubernetes'
    const destinationName = type + '.' + name

		console.log(destinationName)
		updateCurrentDestinationName(name)
    updateEnabledCommandInput(name.length > 0)

    setNumDestinations(destinations.filter((item) => item.name === name).length)
    
    axios.get('/v1/identities?name=connector')
      .then((response) => {
        const { id } = response.data[0]
        const keyName = name + '-' + [...Array(10)].map(() => (~~(Math.random() * 36)).toString(36)).join('')

        return { identityID: id, name: keyName, ttl: '87600h', extensionDeadline: '720h'}
      })
      .then((config) => {
        return axios.post('/v1/access-keys', config)
      })
      .then((accessKeyInfo) => {
        updateAccessKey(accessKeyInfo.data.accessKey)
      })
      .catch((error) => {
        console.log(error)
      })
  }

  return (
    <NameContainer>
      <InputContainer>
        <Input 
          label='Provide a name for your cluster'
          value={name}
          onChange={e => setName(e.target.value)}
        />
      </InputContainer>
      <NextButton disabled={name.length === 0} onClick={() => handleNext()}>Next</NextButton>
    </NameContainer>
  ) 
}

export default NameInput