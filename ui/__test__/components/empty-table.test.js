import React from 'react'
import { render, screen } from '@testing-library/react'
import '@testing-library/jest-dom'

import EmptyTable from '../../components/empty-table'

const EmptyTableProps = {
  title: 'test empty table title',
  subtitle: 'test empty table subtitle',
  iconPath: '/test/iconPath',
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
      />
    )

    expect(screen.getByText(EmptyTableProps.title)).toBeInTheDocument()
    expect(screen.getByText(EmptyTableProps.subtitle)).toBeInTheDocument()
  })
})
