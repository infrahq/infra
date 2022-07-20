import React from 'react'
import { render, screen } from '@testing-library/react'
import '@testing-library/jest-dom'

import PageHeader from '../../components/page-header'

const PageHeaderProps = {
  header: 'test page header',
  buttonLabel: 'test button',
  buttonHref: 'www.infrahq.com',
}

jest.mock(
  'next/link',
  () =>
    ({ children, ...rest }) =>
      React.cloneElement(children, { ...rest })
)

describe('PageHeader Component', () => {
  it('should render', () => {
    expect(() =>
      render(
        <PageHeader
          header={PageHeaderProps.header}
          buttonLabel={PageHeaderProps.buttonLabel}
          buttonHref={PageHeaderProps.buttonHref}
        />
      )
    ).not.toThrow()
  })

  it('should render correct header and button label', () => {
    render(
      <PageHeader
        header={PageHeaderProps.header}
        buttonLabel={PageHeaderProps.buttonLabel}
        buttonHref={PageHeaderProps.buttonHref}
      />
    )

    expect(screen.getByText(PageHeaderProps.header)).toBeInTheDocument()
    expect(screen.getByText(PageHeaderProps.buttonLabel)).toBeInTheDocument()
  })

  it('should render correct button link', () => {
    const { getByTestId } = render(
      <PageHeader
        header={PageHeaderProps.header}
        buttonLabel={PageHeaderProps.buttonLabel}
        buttonHref={PageHeaderProps.buttonHref}
      />
    )

    expect(getByTestId('page-header-button-link')).toHaveAttribute(
      'href',
      PageHeaderProps.buttonHref
    )
  })

  it('should not render button link', () => {
    const { queryByTestId } = render(
      <PageHeader
        header={PageHeaderProps.header}
        buttonLabel={PageHeaderProps.buttonLabel}
      />
    )

    expect(queryByTestId('page-header-button-link')).not.toBeInTheDocument()
  })
})
