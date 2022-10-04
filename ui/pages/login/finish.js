import { useRouter } from 'next/router'
import { useState } from 'react'

import ErrorMessage from '../../components/error-message'
import Login from '../../components/layouts/login'

export default function Finish() {
  const router = useRouter()

  const [password, setPassword] = useState('')
  const [confirmPassword, setConfirmPassword] = useState('')
  const [error, setError] = useState('')
  const [errors, setErrors] = useState('')

  const { query } = router
  const { next, user } = query

  async function finish(e) {
    e.preventDefault()

    if (password !== confirmPassword) {
      setErrors({
        confirmPassword: 'passwords do not match',
      })
      return false
    }

    try {
      const res = await fetch(`/api/users/${user}`, {
        method: 'PUT',
        body: JSON.stringify({ password }),
      })

      if (!res.ok) {
        throw await res.json()
      }

      router.replace(next ? decodeURIComponent(next) : '/')
    } catch (e) {
      if (e.fieldErrors) {
        const errors = {}
        for (const error of e.fieldErrors) {
          errors[error.fieldName.toLowerCase()] =
            error.errors[0] || 'invalid value'
        }
        setErrors(errors)
      } else {
        setError(e.message || 'Invalid password')
      }
    }
    return false
  }

  return (
    <>
      <h1 className='mt-4 font-display text-xl font-semibold leading-snug'>
        Login to Infra
      </h1>
      <h2 className='my-2 text-center text-sm text-gray-500'>
        You&apos;ve used a one time password.
        <br />
        Set your new password to continue.
      </h2>

      <form
        onSubmit={finish}
        className='relative my-4 flex w-full max-w-sm flex-col'
      >
        <div className='my-2 w-full'>
          <label
            htmlFor='password'
            className='text-2xs font-medium text-gray-700'
          >
            New Password
          </label>
          <input
            required
            name='password'
            type='password'
            placeholder='enter your new password'
            onChange={e => {
              setPassword(e.target.value)
              setErrors({})
              setError('')
            }}
            className={`mt-1 block w-full rounded-md shadow-sm focus:border-blue-500 focus:ring-blue-500 sm:text-sm ${
              error ? 'border-red-500' : 'border-gray-300'
            }`}
          />
          {errors.password && <ErrorMessage message={errors.password} />}
        </div>
        <div className='my-2 w-full'>
          <label
            htmlFor='confirm-password'
            className='text-2xs font-medium text-gray-700'
          >
            Confirm Password
          </label>
          <input
            required
            name='confirm-password'
            type='password'
            placeholder='confirm your password'
            onChange={e => {
              setConfirmPassword(e.target.value)
              setErrors({})
              setError('')
            }}
            className={`mt-1 block w-full rounded-md shadow-sm focus:border-blue-500 focus:ring-blue-500 sm:text-sm ${
              error ? 'border-red-500' : 'border-gray-300'
            }`}
          />
          {errors.confirmPassword && (
            <ErrorMessage message={errors.confirmPassword} />
          )}
        </div>
        <button className='mt-4 mb-2 flex w-full cursor-pointer justify-center rounded-md border border-transparent bg-blue-500 py-2 px-4 text-sm font-medium text-white shadow-sm hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2'>
          Log in
        </button>
        {error && <ErrorMessage message={error} />}
      </form>
    </>
  )
}

Finish.layout = page => <Login>{page}</Login>
