import Router from "next/router"
import { useContext } from "react"
import styled, { keyframes } from "styled-components"

import ActionButton from "../ActionButton"

import DestinationsContext from "../../store/DestinationsContext"

const spinner = keyframes`
  0% { transform: rotate(0deg); }
  100% { transform: rotate(360deg); }
`

const ConnectStatusContentContainer = styled.div`
  display: flex;
  flex-direction: row;
  align-items: center;

  & > *:not(:first-child) {
    margin-left: 1rem;
  }
`

const ConnectedIcon = styled.div`
  width: 1rem;
  height: 1rem;
  border-radius: 50%;
  background-color: #008958;
  opacity: .5;
  box-shadow: 
    0 0 30px 2.5px #fff, 
    0 0 40px 3px #f0f, 
    0 0 80px 5px #0ff;
`

const SpinIcon = styled.div`
  margin: auto;
  border: .25rem solid #EAF0F6;
  border-radius: 50%;
  border-top: .25rem solid #008958;
  width: 1rem;
  height: 1rem;
  animation: ${spinner} 4s linear infinite;
`


const ConnectStatus = () => {
  const { enabledCommandInput, connected } = useContext(DestinationsContext)

  const handleFinish = async () => {
		await Router.push({
      pathname: '/destinations/'
    }, undefined, { shallow: true })
	}


  return (
    <div>
      <p>Once you have successfully installed infra we will be able to detect the connection</p>
      {enabledCommandInput && 
        <>
          <h1>Connection Status</h1>
          <ConnectStatusContentContainer>
            <div>
              {connected ? 
              <ConnectedIcon></ConnectedIcon> : 
              <SpinIcon></SpinIcon>
              }
            </div>
            <div>
              {connected ? 
                <p>Connected</p> : 
                <p>No connection detected...</p>
              }
            </div>
          </ConnectStatusContentContainer>
          <ActionButton disabled={!enabledCommandInput && !connected} onClick={() => handleFinish()} value='Finish' />
        </>
      }
    </div>
  )
}

export default ConnectStatus