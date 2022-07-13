import React from 'react'
import { render, screen, fireEvent } from '../../test-utils'
// import userEvent from '@testing-library/user-event'

import Login from './index'

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
      screen.getByText('Welcome back. Login with your credentials')
    ).toBeInTheDocument()
    expect(
      screen.queryByText('or via your identity provider.')
    ).not.toBeInTheDocument()
  })

  // TODO: mock swr
  // it('it renders with multiple providers', () => {
  //   const mockedProviders = {
  //     data: {
  //       item: [
  //         {
  //           id: 0,
  //           name: 'Okta',
  //           kind: 'okta',
  //           url: 'example@okta.com',
  //         },
  //         {
  //           id: 1,
  //           name: 'Azure Active Directory',
  //           kind: 'azure',
  //           url: 'example@azure.com',
  //         },
  //       ],
  //     },
  //   }
  //   jest.spyOn(global, 'fetch').mockImplementation(setupFetchStub(mockedProviders))

  //   render(<Login />)

  //   expect(
  //     screen.getByText('or via your identity provider.')
  //   ).toBeInTheDocument()
  // })

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

  // it('should call onSubmit function when username and password are valid and the login button is clicked', () => {
  //   const login = render(<Login />)

  //   const instance = login.instance()
  //   const spy = jest.spyOn(instance, 'onSubmit')

  //   const usernameInput = screen.getByLabelText('Username or Email')
  //   fireEvent.change(usernameInput, {
  //     target: { value: 'example@infrahq.com' },
  //   })

  //   const passwordInput = screen.getByLabelText('Password')
  //   fireEvent.change(passwordInput, {
  //     target: { value: 'password' },
  //   })

  //   expect(screen.getByLabelText('Username or Email')).toHaveValue(
  //     'example@infrahq.com'
  //   )
  //   expect(screen.getByLabelText('Password')).toHaveValue('password')
  //   expect(screen.getByText('Login').closest('button')).not.toBeDisabled()

  //   userEvent.click(screen.getByText('Login'))
  //   expect(spy).toHaveBeenCalled()
  // })
})

// TEST for <Providers ... />
