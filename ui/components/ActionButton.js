import styled, { css } from 'styled-components'
import PropTypes from 'prop-types'

const StyledButton = styled.button`
  width: ${props => props.size === 'large' ? '24rem' : '12rem'};
  height: ${props => props.size === 'large' ? '2.125rem' : '1.5rem'};
  background: linear-gradient(266.64deg, #CB56FF -53.31%, #4EB2F4 93.79%);
  border-radius: 2px;
  border: none;
  color: #ffffff;
  cursor: ${props => props.disabled ? 'not-allowed' : 'pointer'};
  font-size: 10px;
  font-weight: 100;
  opacity: ${props => props.disabled ? 0.5 : 1};

  ${props => !props.disabled && css`  
    &:hover {
      opacity: .95;
    }
  `}
`

const ActionButton = ({ value, onClick, size = 'large', disabled = false }) => {
  return (
    <section>
      <StyledButton disabled={disabled} onClick={onClick} size={size}>{value}</StyledButton>
    </section>
  )
}

ActionButton.prototype = {
  value: PropTypes.string.isRequired,
  onClick: PropTypes.func.isRequired,
  size: PropTypes.oneOf(['large', 'small']),
  disabled: PropTypes.bool
}

export default ActionButton
