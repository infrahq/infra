import React from 'react'
import { render, screen, fireEvent } from '@testing-library/react'
import '@testing-library/jest-dom'

import Signup from '../pages/signup/index'

describe('Signup Component', () => {
  it('should render', () => {
    expect(() => render(<Signup />)).not.toThrow()
  })

  it('should render with correct state', () => {
    render(<Signup />)
    expect(screen.getByText('Welcome to Infra')).toBeInTheDocument()
    expect(screen.getByLabelText('Email')).toHaveValue('')
    expect(screen.getByLabelText('Password')).toHaveValue('')
    expect(screen.getByLabelText('Confirm Password')).toHaveValue('')
    expect(screen.getByLabelText('Organization')).toHaveValue('')
    expect(screen.getByText('Get Started').closest('button')).toBeDisabled()
  })
})
