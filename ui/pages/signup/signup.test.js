import React from 'react'
import { render } from '@testing-library/react'

import Signup from '.'

describe('Signup Component', () => {
  it('should render', () => {
    expect(() => render(<Signup />)).not.toThrow()
  })
})
