import React from 'react'
import { render, screen } from '@testing-library/react'
import '@testing-library/jest-dom'

import Pagination, { Pages, Arrow } from '../../components/pagination'

jest.mock(
  'next/link',
  () =>
    ({ children, ...rest }) =>
      React.cloneElement(children, { ...rest })
)

jest.mock('next/router', () => ({
  useRouter() {
    return {
      pathname: '/users',
    }
  },
}))

describe('Pagination Component', () => {
  const PaginationProps = { curr: 2, totalPages: 4, totalCount: 51 }
  it('should render', () => {
    expect(() =>
      render(
        <Pagination
          curr={1}
          totalPages={PaginationProps.totalPages}
          totalCount={PaginationProps.totalCount}
        />
      )
    ).not.toThrow()
  })

  it('should render correct range text', () => {
    render(
      <Pagination
        curr={PaginationProps.curr}
        totalPages={PaginationProps.totalPages}
        totalCount={PaginationProps.totalCount}
      />
    )
    expect(
      screen.getByText(`Displaying 14–26 out of ${PaginationProps.totalCount}`)
    ).toBeInTheDocument()
  })

  it('should render correct arrow links', () => {
    const path = '/users'
    const { getByTestId } = render(
      <Pagination
        curr={PaginationProps.curr}
        totalPages={PaginationProps.totalPages}
        totalCount={PaginationProps.totalCount}
      />
    )

    expect(getByTestId('LEFT-arrow-button-link')).toHaveAttribute(
      'href',
      path + '?p=' + (PaginationProps.curr - 1)
    )
    expect(getByTestId('RIGHT-arrow-button-link')).toHaveAttribute(
      'href',
      path + '?p=' + (PaginationProps.curr + 1)
    )
  })
})

describe('Pages Component', () => {
  it('should render', () => {
    expect(() =>
      render(
        <Pages path='/users' selected={2} count={7} totalPages={12} />
      ).not.toThrow()
    )
  })

  it('should render correct button link', () => {
    const path = '/users'
    const page = 3
    const { getAllByTestId } = render(
      <Pages path={path} selected={2} count={7} totalPages={12} />
    )

    expect(getAllByTestId('pages-button-link')[2]).toHaveAttribute(
      'href',
      path + '?p=' + page
    )
  })

  it('should render correct button count (> count)', () => {
    const count = 7
    const { getAllByTestId } = render(
      <Pages path='' selected={2} count={count} totalPages={12} />
    )
    const buttons = getAllByTestId('pages-button-link')
    expect(buttons.length).toBe(count)
  })

  it('should render correct button count (< count)', () => {
    const totalPages = 5
    const { getAllByTestId } = render(
      <Pages path='' selected={2} count={7} totalPages={totalPages} />
    )
    const buttons = getAllByTestId('pages-button-link')
    expect(buttons.length).toBe(totalPages)
  })
})

describe('Arrows', () => {
  it('should render', () => {
    expect(() => render(<Arrow path='' direction='LEFT'></Arrow>).not.toThrow())

    expect(() =>
      render(<Arrow path='' direction='RIGHT'></Arrow>).not.toThrow()
    )
  })

  it('should render passed path', () => {
    const path = 'this is a test'
    let { getByTestId } = render(<Arrow path={path} direction='LEFT'></Arrow>)

    expect(getByTestId('LEFT-arrow-button-link')).toHaveAttribute('href', path)
  })

  it('should render left arrow', () => {
    let { getByTestId } = render(<Arrow direction='LEFT'></Arrow>)
    expect(getByTestId('left-arrow')).toBeInTheDocument()
  })

  it('should render right arrow', () => {
    let { getByTestId } = render(<Arrow direction='RIGHT'></Arrow>)
    expect(getByTestId('right-arrow')).toBeInTheDocument()
  })
})
