import Link from 'next/link'
import styled from 'styled-components'

const ExitButtonContainer = styled.div`
  cursor: pointer;

  &:hover {
    opacity: .5
  }
`

const ExitButton = () => {
  return (
    <ExitButtonContainer>
      <Link href='/'>
        <img src='/close-icon.svg' />
      </Link>
    </ExitButtonContainer>
  )
}

export default ExitButton
