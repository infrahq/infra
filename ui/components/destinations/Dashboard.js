import { useContext } from "react"
import axios from 'axios'
import Router from 'next/router'
import styled from 'styled-components'

import FormattedTime from "../FormattedTime"
import EmptyPageHeader from "../EmptyPageHeader"

import DestinationsContext from "../../store/DestinationsContext"

const TableContent = styled.div`
  display: grid;
  grid-template-columns: 80% 20%;
  align-items: center;
`

const TableContentText = styled.div`
  font-weight: 300;
  font-size: 12px;
  line-height: 0px;

  a {
    cursor: pointer;

    :hover {
      opacity: .6;
    }
  }
`

const TableContentContainer = styled.div`
  padding-top: 1rem;
`

const Dashboard = () => {
	const { destinations } = useContext(DestinationsContext);

  const handleAddDestination = async () => {
    await Router.push({
      pathname: '/destinations/add/connect'
    }, undefined, { shallow: true })
  }

	return (
		<div>
			{destinations.length > 0
				? (
					<TableContentContainer>
						{destinations.map((item) => {
							return (
								<TableContent key={item.id}>
									<TableContentText>{item.name}</TableContentText>
									<TableContentText>
										<FormattedTime time={item.created} />
									</TableContentText>
								</TableContent>
							)
						})}
					</TableContentContainer>
				)
			: (
				<EmptyPageHeader
					header='Destinations'
					subheader='No destinations connected.'
					actionButtonHeader='Add Destinations'
					onClickActionButton={() => handleAddDestination()}
				/>
			)
			}
		</div>
	)
}

export default Dashboard