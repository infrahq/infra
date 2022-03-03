import Link from 'next/link'
import styled from 'styled-components'

import Input from '../../Input'
import Header from '../../Header'
import Logo from './Logo'

const HelpContainer = styled.div`
  font-weight: 100;
  font-size: 11px;
  line-height: 13px;
  max-width: 24rem;

  span {
    opacity: .5;
  }

  a {
    padding-left: .5rem;
    color: #93DEFF;
    text-decoration: none;

    :hover {
      opacity: .95;
      text-decoration: underline;
    }
  }
`

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
      <Logo />
      <Header
        header='Set up an infra admin'
        subheader={
          <HelpContainer>
            <span>Great work. Now provide an email of your Okta users that you would like to make an Infra admistrator.</span>
            <a
              target='_blank'
              rel='noreferrer'
              href='https://github.com/infrahq/infra/blob/main/docs/providers/okta.md'
            >
              Make sure to assign them in Okta.
            </a>
          </HelpContainer>
        }
      />
      <div>
        <Input
          label='Admin Email'
          value={email}
          onChange={(e) => parentCallback(e.target.value)}
        />
      </div>
      <Link href='/'>
        <StyledLink>
          <a>Or set up an Infra admin later.</a>
        </StyledLink>
      </Link>
    </>
  )
}

export default AddAdmin
