import React from 'react'
import { render, screen, fireEvent } from '@testing-library/react'
import Login from './index'

describe('Login Component', () => {
  test('it renders', () => {
    render(<Login />)

    expect(() => render(<Login />)).not.toThrow()
  })

  test('it renders with correct state', () => {
    render(<Login />)

    expect(screen.getByText('Login to Infra')).toBeInTheDocument()
    expect(screen.getByLabelText('Username or Email')).toHaveValue('')
    expect(screen.getByLabelText('Password')).toHaveValue('')
    expect(screen.getByText('Login').closest('button')).toBeDisabled()
  })

  test('it renders with no provider', () => {
    render(<Login />)

    expect(
      screen.getByText('Welcome back. Login with your credentials')
    ).toBeInTheDocument()
    expect(
      screen.queryByText('or via your identity provider.')
    ).not.toBeInTheDocument()
  })

  // TODO: mock swr
  // test('it renders with multiple providers', () => {
  //   render(<Login />)

  //   expect(
  //     screen.getByText('or via your identity provider.')
  //   ).toBeInTheDocument()
  // })

  test('it enters username', () => {
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

  test('it enters both username and password', () => {
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

  // test('it redirects to the dashboard when submits a valid username and password', () => {
  //   render(<Login />)

  //   // type username and password
  //   expect(screen.getByLabelText('Username or Email')).toHaveValue('')
  //   expect(screen.getByLabelText('Password')).toHaveValue('')
  //   expect(screen.getByText('Login').closest('button')).not.toBeDisabled()
  // })

  // test('it has error message when submit an invalid username and password', () => {
  //   render(<Login />)

  //   // type username and password
  //   expect(screen.getByLabelText('Username or Email')).toHaveValue('')
  //   expect(screen.getByLabelText('Password')).toHaveValue('')
  //   expect(screen.getByText('Login').closest('button')).not.toBeDisabled()
  // })
})
