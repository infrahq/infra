import React from 'react'
import { render, screen, fireEvent } from '@testing-library/react'
import useSWR from 'swr'
import '@testing-library/jest-dom'

import Login from './index'

function mockedProviders() {
  return {
    data: {
      items: [],
    },
  }
}

jest.mock('swr', () => {
  return {
    __esModule: true,
    default: jest.fn(mockedProviders),
    useSWRConfig: jest.fn(() => ({
      mutate: () => {},
    })),
  }
})

describe('Login Component', () => {
  it('should render', () => {
    expect(() => render(<Login />)).not.toThrow()
  })

  it('should render with correct state', () => {
    render(<Login />)

    expect(screen.getByText('Login to Infra')).toBeInTheDocument()
    expect(screen.getByLabelText('Username or Email')).toHaveValue('')
    expect(screen.getByLabelText('Password')).toHaveValue('')
    expect(screen.getByText('Login').closest('button')).toBeDisabled()
  })

  it('should render with no provider', () => {
    render(<Login />)

    expect(
      screen.queryByText('Welcome back. Login with your credentials')
    ).toBeInTheDocument()
    expect(
      screen.queryByText('or via your identity provider.')
    ).not.toBeInTheDocument()
  })

  it('it renders with multiple providers', () => {
    const providers = {
      data: {
        items: [
          {
            id: 0,
            name: 'Okta',
            kind: 'okta',
            url: 'example@okta.com',
          },
          {
            id: 1,
            name: 'Azure Active Directory',
            kind: 'azure',
            url: 'example@azure.com',
          },
        ],
      },
    }

    useSWR.mockReturnValue(providers)

    render(<Login />)

    expect(
      screen.getByText(
        'Welcome back. Login with your credentials or via your identity provider.'
      )
    ).toBeInTheDocument()
  })

  it('should not enable the login button when enter username only', () => {
    render(<Login />)

    const usernameInput = screen.getByLabelText('Username or Email')
    fireEvent.change(usernameInput, {
      target: { value: 'example@infrahq.com' },
    })
    expect(screen.getByLabelText('Username or Email')).toHaveValue(
      'example@infrahq.com'
    )
    expect(screen.getByLabelText('Password')).toHaveValue('')
    expect(screen.getByText('Login').closest('button')).toBeDisabled()
  })

  it('should enable the login button when enter both username and password', () => {
    render(<Login />)

    const usernameInput = screen.getByLabelText('Username or Email')
    fireEvent.change(usernameInput, {
      target: { value: 'example@infrahq.com' },
    })

    const passwordInput = screen.getByLabelText('Password')
    fireEvent.change(passwordInput, {
      target: { value: 'password' },
    })

    expect(screen.getByLabelText('Username or Email')).toHaveValue(
      'example@infrahq.com'
    )
    expect(screen.getByLabelText('Password')).toHaveValue('password')
    expect(screen.getByText('Login').closest('button')).not.toBeDisabled()
  })
})

// TEST for <Providers ... />
// describe('Providers Component', () => {
//   it('should render when there is provider')
// })
