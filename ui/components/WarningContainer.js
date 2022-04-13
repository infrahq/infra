import styled from 'styled-components'
import PropTypes from 'prop-types'

const WarningSection = styled.div`
  display: flex;
  flex-direction: row;
  height: 58px;
  width: auto;
  background: rgba(255, 255, 255, 0.02);
  border-left: 2px solid #808EF9;
  box-sizing: border-box;
  box-shadow: 0px 4px 4px rgba(0, 0, 0, 0.25);
  border-radius: 2px;

  & > *:not(:first-child) {
    padding-left: 1rem;
  }
`

const WarningImg = styled.img`
  width: 15.17px;
  height: 13.92px;
  padding-top: 21px;
  padding-left: 20px;
`

const WarningContentText = styled.div`
  font-style: normal;
  font-weight: 400;
  font-size: 11px;
  line-height: 156.52%;
  color: #FFFFFF;
  opacity: 0.5;
  padding-top: 1rem;
`

const WarningContainer = ({ text }) => {
  return (
    <div>
      <WarningSection>
        <WarningImg src='/warning-icon.svg' />
        <WarningContentText>{text}</WarningContentText>
      </WarningSection>
    </div>
  )
}

WarningContainer.prototype = {
  text: PropTypes.string.isRequired
}

export default WarningContainer
