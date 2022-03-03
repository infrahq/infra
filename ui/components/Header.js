import styled from 'styled-components'

const Container = styled.section`
  & > *:not(:first-child) {
    padding-top: .5rem;
  }
`

const StyledHeader = styled.div`
  font-weight: 200;
  font-size: 10px;
  line-height: 4px;

  display: flex;
  align-items: center;
  text-transform: uppercase;
`

const StyledSubheader = styled.div`
  font-weight: 100;
  font-size: 10px;
  line-height: 12px;
  display: flex;
  align-items: center;

  opacity: 0.4;
`

const Header = ({ header, subheader }) => {
  return (
    <Container>
      <StyledHeader>{header}</StyledHeader>
      <StyledSubheader>{subheader}</StyledSubheader>
    </Container>
  )
}

export default Header
