import { createContext, useEffect, useState } from "react";
import axios from 'axios'

const DestinationsContext = createContext({
	destinations: [],
	updateDestinationsList: () => {}
})

export const DestinationsContextProvider = ({ children }) => {
	const [destinations, setDestinations] = useState([])

	useEffect(() => {
    const source = axios.CancelToken.source()
    axios.get('/v1/destinations')
			.then((response) => {
				console.log(response)
				setDestinations(response.data)
			})
			.catch((error) => {
				console.log(error)
			})
    return function () {
      source.cancel('Cancelling in cleanup')
    }
  }, [])

	const updateDestinationsList = (list) => {
		setDestinations(list)
	}

	const context = {
		destinations,
		updateDestinationsList
	}

	return (
		<DestinationsContext.Provider value={context}>
			{children}
		</DestinationsContext.Provider>
	)
}

export default DestinationsContext