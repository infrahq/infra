import React from 'react'
import { render, screen, fireEvent } from '@testing-library/react'

import Badge from '../../components/badge'

describe('Badge Component', () => {
  it('should render', () => {
    expect(() => render(<Badge onRemove={() => {}} />)).not.toThrow()
  })

  it('should render with correct text', () => {
    const children = 'badge test children'
    render(<Badge onRemove={() => {}}>{children}</Badge>)

    expect(screen.getByText(children)).toBeInTheDocument()
  })

  it('should fires onRemove when clicks remove button', () => {
    const children = 'badge test children'
    const handleOnRemove = jest.fn()
    render(<Badge onRemove={handleOnRemove}>{children}</Badge>)

    fireEvent.click(screen.getByTestId('badgeRemoveIcon'))
    expect(handleOnRemove).toHaveBeenCalledTimes(1)
  })
})
