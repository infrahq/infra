import styled from 'styled-components';

interface Button {
  children: React.ReactNode;
  onClick: () => void;
  size?: 'large' | 'small';
};

const StyledButton = styled.button<Button>`
  width: ${props => props.size === 'large' ? '24rem' : '12rem'};
  height: ${props => props.size === 'large' ? '2.125rem' : '1.5rem'};
  background: linear-gradient(266.64deg, #CB56FF -53.31%, #4EB2F4 93.79%);
  border-radius: 2px;
  border: none;
  color: #ffffff;
  cursor: pointer;

  &:hover {
    opacity: .95;
  }
`;


const ActionButton = ({children, onClick, size = 'large'}: Button) => {
  return (
    <StyledButton onClick={onClick} size={size}>{children}</StyledButton>
  )
};

export default ActionButton; 