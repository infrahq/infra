import { createContext, useEffect, useState } from "react";
import axios from 'axios'

const DestinationsContext = createContext({
	destinations: [],
	currentDestinationName: null,
	accessKey: null,
	updateDestinationsList: () => {},
	updateCurrentDestinationName: () => {},
	updateAccessKey: () => {}
})

export const DestinationsContextProvider = ({ children }) => {
	const [destinations, setDestinations] = useState([])
	const [currentDestinationName, setCurrentDestinationName] = useState(null)
	const [accessKey, setAccessKey] = useState(null)

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

	const updateCurrentDestinationName = (name) => {
		setCurrentDestinationName(name)
	}

	const updateAccessKey = (key) => {
		setAccessKey(key)
	}

	const context = {
		destinations,
		currentDestinationName,
		accessKey,
		updateDestinationsList,
		updateCurrentDestinationName,
		updateAccessKey
	}

	return (
		<DestinationsContext.Provider value={context}>
			{children}
		</DestinationsContext.Provider>
	)
}

export default DestinationsContext