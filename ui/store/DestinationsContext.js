import { createContext, useEffect, useState } from "react";
import axios from 'axios'

const DestinationsContext = createContext({
	destinations: [],
	getDestinations: () => {}
})

export const DestinationsContextProvider = ({ children }) => {
	const [destinations, setDestinations] = useState([])

	const getDestinations = () => {
		axios.get('/v1/destinations')
			.then((response) => {
				console.log(response)
				const destinationsList = response.data
				setDestinations(destinationsList)
				return destinationsList
			})
			.catch((error) => {
				console.log(error)
			})
	}

	const context = {
		destinations,
		getDestinations
	}

	return (
		<DestinationsContext.Provider value={context}>
			{children}
		</DestinationsContext.Provider>
	)
}

export default DestinationsContext