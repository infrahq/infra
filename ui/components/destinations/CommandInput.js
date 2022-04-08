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
  white-space: pre;
`

const CommandInput = () => {
  const { enabledCommandInput, accessKey, currentDestinationName } = useContext(DestinationsContext)

  const value = enabledCommandInput ? `helm install infra-connector infrahq/infra \\
  --set connector.config.accessKey=${accessKey} \\
  --set connector.config.server=${window.location.host} \\
  --set connector.config.name=${currentDestinationName}` : ''

  return (
    <section>
      <p>Run the following command to connect your cluster</p>
      <CommandInputTextAreaContainer readOnly value={value} />
    </section>
  )
}

export default CommandInput