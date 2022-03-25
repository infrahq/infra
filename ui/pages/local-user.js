import styled from 'styled-components'

import Nav from '../components/nav/Nav'
import PageHeader from '../components/PageHeader'

const Container = styled.section`
  display: grid;
  column-gap: 2rem;
  grid-template-columns: 18% auto;
`

const LocalUser = () => {
  return (
    <Container>
      <Nav />
      <div>
        <PageHeader iconPath='/local-users.svg' title='Local Users' />
      </div>
    </Container>
  )
}

export default LocalUser
