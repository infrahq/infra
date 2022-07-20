import React from 'react'
import { render, screen } from '@testing-library/react'

import ErrorMessage from '../../components/error-message'

const mockedMessage = 'test error message'

describe('Error Message Component', () => {
  it('should render', () => {
    expect(() => render(<ErrorMessage message={mockedMessage} />)).not.toThrow()
  })

  it('should render with correct error message', () => {
    render(<ErrorMessage message={mockedMessage} />)

    expect(screen.getByText(mockedMessage)).toBeInTheDocument()
  })
})
