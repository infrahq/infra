import styled from "styled-components"
import PropTypes from 'prop-types'

const PageHeaderContainer = styled.section`
  display: flex;
  flex-direction: row;
  align-items: center;

  & > *:not(:first-child) {
    padding-left: 1.375rem;
  }
`

const PageHeaderTitle = styled.div`
  font-style: normal;
  font-weight: 400;
  font-size: 11px;
  line-height: 0%;
  text-transform: uppercase;
`

const PageHeader = ({ iconPath, title }) => {
  return (
    <PageHeaderContainer>
      <img src={iconPath} />
      <PageHeaderTitle>{title}</PageHeaderTitle>
    </PageHeaderContainer>
  )
}

PageHeader.prototype = {
  iconPath: PropTypes.string.isRequired,
  title: PropTypes.string.isRequired
}

export default PageHeader