import PropTypes from 'prop-types'
import styled from 'styled-components'

const InputContainer = styled.section`
  position: relative;
`

const InputGroup = styled.div`
  opacity: 0.5;
  border: 1px solid rgba(255, 255, 255, 0.1);
  box-sizing: border-box;
  border-radius: 2px;
`

const StyledInputContainer = styled.div`
  display: flex;
  flex-direction: row;
  justify-content: space-between;
  padding: 0 .5rem 0 .75rem;
`

const StyledInput = styled.input.attrs({
  type: 'text'
})`
  border: none;
  background: transparent;
  width: 20.75rem;
  height: 2.125rem;
  color: #ffffff;

  &:focus-visible {
    outline: 0;
  }
`

const Label = styled.span`
  position: absolute;
  width: 6.625rem;
  height: .75rem;
  left: .625rem;
  top: -5px;
  padding: 0 .375rem;
  background-color: #0A0E12;
  font-weight: 100;
  font-size: .625rem;
  line-height: .75rem;
  color: rgba(255, 255, 255, 0.5);
`

const Input = ({ label, value, onChange, showImage=false }) => {
  return (
    <InputContainer>
      <InputGroup>
        <Label>{label}</Label>
        <StyledInputContainer>
          <StyledInput
            value={value}
            onChange={onChange}
          />
          {showImage ? <img src='/accessKeyLockIcon.svg' /> : <></>}
        </StyledInputContainer>

      </InputGroup>
    </InputContainer>
  )
}

Input.prototype = {
  label: PropTypes.string.isRequired,
  value: PropTypes.string.isRequired,
  onChange: PropTypes.func.isRequired,
  showImage: PropTypes.bool
}

export default Input
