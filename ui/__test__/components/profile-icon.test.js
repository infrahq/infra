import { render, screen } from '@testing-library/react'
import React from 'react'

import ProfileIcon from '../../components/profile-icon'

describe('Profile Icon Component', () => {
  it('should render', () => {
    expect(() => render(<ProfileIcon name='test profile name' />)).not.toThrow()
  })

  it('should render with correct name', () => {
    const name = 'test profile icon'
    render(<ProfileIcon name={name} />)

    expect(screen.getByText(name)).toBeInTheDocument()
  })
})
