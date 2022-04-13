import styled from 'styled-components'

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

const CommandInput = ({ enabledCommandInput, accessKey, currentDestinationName }) => {
  const server = window.location.host
  const isHttps = window.location.origin.includes('https')
  const defaultValue = `helm install infra-connector infrahq/infra \\
  --set connector.config.accessKey=${accessKey} \\
  --set connector.config.server=${server} \\
  --set connector.config.name=${currentDestinationName}`

  const commandValue = isHttps
    ? defaultValue
    : defaultValue + ` \\
  --set connector.config.skipTLSVerify=true`

  const value = enabledCommandInput ? commandValue : ''

  return (
    <section>
      <p>Run the following command to connect your cluster</p>
      <CommandInputTextAreaContainer readOnly value={value} />
    </section>
  )
}

export default CommandInput
