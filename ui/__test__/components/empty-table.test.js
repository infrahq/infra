import React from 'react'
import { render, screen } from '@testing-library/react'
import '@testing-library/jest-dom'

import EmptyTable from '../../components/empty-table'

const EmptyTableProps = {
  title: 'test empty table title',
  subtitle: 'test empty table subtitle',
  iconPath: '/test/iconPath',
  buttonText: 'test button',
  buttonHref: 'www.infrahq.com',
}

jest.mock(
  'next/link',
  () =>
    ({ children, ...rest }) =>
      React.cloneElement(children, { ...rest })
)

describe('Empty Table Component', () => {
  it('should render', () => {
    expect(() =>
      render(
        <EmptyTable
          title={EmptyTableProps.title}
          subtitle={EmptyTableProps.subtitle}
          iconPath={EmptyTableProps.iconPath}
          buttonText={EmptyTableProps.buttonText}
          buttonHref={EmptyTableProps.buttonHref}
        />
      )
    ).not.toThrow()
  })

  it('should render correct title, subtitle and button label', () => {
    render(
      <EmptyTable
        title={EmptyTableProps.title}
        subtitle={EmptyTableProps.subtitle}
        iconPath={EmptyTableProps.iconPath}
        buttonText={EmptyTableProps.buttonText}
        buttonHref={EmptyTableProps.buttonHref}
      />
    )

    expect(screen.getByText(EmptyTableProps.title)).toBeInTheDocument()
    expect(screen.getByText(EmptyTableProps.subtitle)).toBeInTheDocument()
    expect(screen.getByText(EmptyTableProps.buttonText)).toBeInTheDocument()
  })

  it('should render correct icon path', () => {
    const { getByAltText } = render(
      <EmptyTable
        title={EmptyTableProps.title}
        subtitle={EmptyTableProps.subtitle}
        iconPath={EmptyTableProps.iconPath}
        buttonText={EmptyTableProps.buttonText}
        buttonHref={EmptyTableProps.buttonHref}
      />
    )

    const image = getByAltText(EmptyTableProps.title)

    expect(image).toHaveAttribute('src', EmptyTableProps.iconPath)
  })

  it('should render correct button link', () => {
    const { getByTestId } = render(
      <EmptyTable
        title={EmptyTableProps.title}
        subtitle={EmptyTableProps.subtitle}
        iconPath={EmptyTableProps.iconPath}
        buttonText={EmptyTableProps.buttonText}
        buttonHref={EmptyTableProps.buttonHref}
      />
    )

    expect(getByTestId('empty-table-button-link')).toHaveAttribute(
      'href',
      EmptyTableProps.buttonHref
    )
  })

  it('should not render button link', () => {
    const { queryByTestId } = render(
      <EmptyTable
        title={EmptyTableProps.title}
        subtitle={EmptyTableProps.subtitle}
        iconPath={EmptyTableProps.iconPath}
        buttonText={EmptyTableProps.buttonText}
      />
    )

    expect(queryByTestId('empty-table-button-link')).not.toBeInTheDocument()
  })
})
