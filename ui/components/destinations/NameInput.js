import { useContext, useState } from "react"
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
  const { updateCurrentDestinationName } = useContext(DestinationsContext)
  const [name, setName] = useState('')

  const handleNext = () => {
    const type = 'kubernetes'
    const destinationName = type + '.' + name

		console.log(destinationName)
		updateCurrentDestinationName(name)

    axios.get('/v1/identities')
      .then((response) => {
        console.log(response)
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
      <NextButton onClick={() => handleNext()}>Next</NextButton>
    </NameContainer>
  ) 
}

export default NameInput