import Link from 'next/link'
import styled from 'styled-components'

import Input from "../../Input"
import Header from "../../Header"

const StyledLink = styled.div`
  font-weight: 100;
  font-size: 11px;
  line-height: 13px;
  max-width: 24rem;

  text-align: center;

  a {
    padding-left: .5rem;
    color: #93DEFF;
    text-decoration: none;
    cursor: pointer;

    :hover {
      opacity: .95;
      text-decoration: underline;
    }
  }
`

const AddAdmin = ({ email, parentCallback }) => {
  return (
    <>
      <Header
        header='Assign an infra admin'
        subheader='Great work. Now set up your infra admininstrator.'
      />
      <img src='/adminCard.svg' />
      <div>
        <Input 
          label='Admin Email'
          value={email}
          onChange={(e) => parentCallback(e.target.value)}
        />
      </div>
      <Link  href='/'>
        <StyledLink>
          <a>Or set up an Infra admin later.</a>
        </StyledLink>
      </Link>
    </>
  )
}

export default AddAdmin