import styled from 'styled-components';

interface IdentitySourceBtnField {
  disabled: boolean;
  type: string;
  onClick?: () => void;
}

const IdentitySourceContainer = styled.button`
  width: 373px;
  height: 45px;
  background: rgba(255,255,255,0.02);
  opacity: ${props => props.disabled ? '.56' : '1'};
  border-radius: 4px;
  border: none;
  cursor: ${props => props.disabled ? 'default' : 'pointer'};
  color: #FFFFFF;

  ${ props => props.disabled 
    ? '' 
    : '&:hover { opacity: .95 }'
  }
`;

const IdentitySourceContentContainer = styled.div`
  display: flex;
  flex-direction: row;
  padding: 6px 7px;
`

const IdentitySourceLogo = styled.div`
  padding-top: .4rem;  
`;

const IdentitySourceContentDescriptionContainer = styled.div`
  padding-left: 1rem;
  text-align: left;
`;

const DescriptionHeader = styled.div`
  font-weight: 100;
  font-size: 12px;
  line-height: 15px;
  text-transform: capitalize;
`;

const DescriptionSubheader = styled.div`
  font-weight: 100;
  font-size: 9px;
  line-height: 11px;
  text-transform: uppercase;
  color: #FFFFFF;
  opacity: 0.3;
`;

const IdentitySourceBtn = ({ type, disabled, onClick }: IdentitySourceBtnField ) => {
  return (
    <IdentitySourceContainer
      onClick={disabled ? undefined : onClick }
      disabled={disabled}
    >
      <IdentitySourceContentContainer>
        <IdentitySourceLogo>
          <img src={`/${type}.svg`} />
        </IdentitySourceLogo>
        <IdentitySourceContentDescriptionContainer>
          <DescriptionHeader>{type}</DescriptionHeader>
          <DescriptionSubheader>Identity Source</DescriptionSubheader>
        </IdentitySourceContentDescriptionContainer>
      </IdentitySourceContentContainer>
    </IdentitySourceContainer>
  )
};

export default IdentitySourceBtn;