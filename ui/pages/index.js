import styled from 'styled-components'

import Nav from '../components/nav/Nav'
import PageHeader from '../components/PageHeader'

const Container = styled.section`
  display: grid;
  column-gap: 2rem;
  grid-template-columns: 18% auto;
`

export default function Index () {
  return (
    <Container>
      <Nav />
      <div>
        <PageHeader iconPath='/access.svg' title='Access' />
      </div>
    </Container>
  )
}
