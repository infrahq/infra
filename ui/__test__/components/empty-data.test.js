import React from 'react'
import { render, screen } from '@testing-library/react'

import EmptyData from '../../components/empty-data'

describe('Badge Component', () => {
  it('should render', () => {
    expect(() => render(<EmptyData />)).not.toThrow()
  })

  it('should render with correct text', () => {
    const children = 'empty data text'
    render(<EmptyData>{children}</EmptyData>)

    expect(screen.getByText(children)).toBeInTheDocument()
  })
})
