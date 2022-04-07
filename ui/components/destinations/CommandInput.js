import { useContext } from "react"
import styled from "styled-components"

import DestinationsContext from "../../store/DestinationsContext"

const CommandInputTextAreaContainer = styled.textarea`
  width: 24rem;
  height: 6.5rem;
  padding: 1rem .75rem;
  background: transparent;
  color: white;
  border: 1px solid rgba(255,255,255,0.25);
  box-sizing: border-box;
  border-radius: 1px;
  resize: none;
`

const CommandInput = () => {
  const { enabledCommandInput, accessKey, currentDestinationName } = useContext(DestinationsContext)

  const server = 'todo'; // TODO: how to get server?
  const value = enabledCommandInput ? `helm install infra-connector infrahq/infra \\
  --set connector.config.accessKey=${accessKey}
  --set connector.config.server=${server}
  --set connector.config.name=${currentDestinationName}` : ''

  return (
    <section>
      <p>Run the following command to connect your cluster</p>
      <CommandInputTextAreaContainer readOnly value={value} />
    </section>
  )
}

export default CommandInput