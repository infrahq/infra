import { useContext } from "react"

import DestinationsContext from "../../store/DestinationsContext"

const ConnectStatus = () => {
  const { enabledCommandInput, connected } = useContext(DestinationsContext)

  return (
    <div>
      {/* <p>Once you have successfully installed infra we will be able to detect the connection</p>
      <ConnectStatusTitle>Connection Status</ConnectStatusTitle>
      <ConnectStatusContentContainer>
        
      </ConnectStatusContentContainer> */}
    </div>
  )
}

export default ConnectStatus