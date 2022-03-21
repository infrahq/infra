import PropTypes from 'prop-types'
import styled from 'styled-components'

import ActionButton from '../components/ActionButton'

const EmptyPageHeaderContainer = styled.div`
  margin-top: 3.5rem;
`

const StyledHeader = styled.div`
  font-style: normal;
  font-weight: 400;
  font-size: 22px;
  line-height: 27px;
  letter-spacing: -0.035em;
  padding-bottom: 1rem;
`

const StyledSubheader = styled.div`
  font-style: normal;
  font-weight: 400;
  font-size: 11px;
  line-height: 13px;
  padding-bottom: 3.5rem;
`

const EmptyPageHeader = ({ header, subheader, actionButtonHeader, onClickActionButton }) => {
  return (
    <EmptyPageHeaderContainer>
      <StyledHeader>{header}</StyledHeader>
      <StyledSubheader>{subheader}</StyledSubheader>
      <ActionButton
        onClick={() => onClickActionButton()}
        value={actionButtonHeader}
        size='small'
      />
    </EmptyPageHeaderContainer>
  )
}

EmptyPageHeader.prototype = {
  header: PropTypes.string.isRequired,
  subheader: PropTypes.string.isRequired,
  actionButtonHeader: PropTypes.string.isRequired,
  onClickActionButton: PropTypes.func.isRequired
}

export default EmptyPageHeader
