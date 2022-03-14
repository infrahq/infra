import Link from 'next/link'
import styled from 'styled-components'

const ExitBtnContainer = styled.div`
  cursor: pointer;

  &:hover {
    opacity: .5
  }
`

const ExitButton = () => {
  return (
    <ExitBtnContainer>
      <Link href='/'>
        <img src='/close-icon.svg' />
      </Link>
    </ExitBtnContainer>
  )
}

export default ExitButton
