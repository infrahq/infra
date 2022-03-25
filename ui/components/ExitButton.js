import PropTypes from 'prop-types'
import Link from 'next/link'
import styled from 'styled-components'

const ExitButtonContainer = styled.div`
  cursor: pointer;

  &:hover {
    opacity: .5
  }
`

const ExitButton = ({ previousPage = '/' }) => {
  return (
    <ExitButtonContainer>
      <Link href={previousPage}>
        <img src='/close-icon.svg' />
      </Link>
    </ExitButtonContainer>
  )
}

ExitButton.prototype = {
  previousPage: PropTypes.string
}

export default ExitButton
