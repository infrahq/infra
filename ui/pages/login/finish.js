import { useRouter } from 'next/router'
import { useState } from 'react'
import { useSWRConfig } from 'swr'

import ErrorMessage from '../../components/error-message'
import Login from '../../components/layouts/login'

export default function Finish() {
  const router = useRouter()

  const [password, setPassword] = useState('')
  const [confirmPassword, setConfirmPassword] = useState('')
  const [error, setError] = useState('')
  const [errors, setErrors] = useState('')

  const { mutate } = useSWRConfig()

  const { query } = router
  const { user } = query

  if (!user) {
    router.replace('/login')
  }

  async function finish(e) {
    e.preventDefault()

    if (password !== confirmPassword) {
      setErrors({
        confirmPassword: 'the confirm password confirmation does not match.',
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

      await mutate('/api/users/self')

      await router.replace('/')
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
      <h1 className='text-base font-bold leading-snug'>Login to Infra</h1>
      <h2 className='my-3 max-w-[260px] text-center text-xs text-gray-300'>
        You&apos;ve used a one time password.
        <br />
        Set your new password to continue.
      </h2>

      <form
        onSubmit={finish}
        className='relative flex w-full max-w-sm flex-col'
      >
        <div className='my-2 w-full'>
          <label
            htmlFor='password'
            className='text-3xs uppercase text-gray-500'
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
            className={`mb-1 w-full border-b border-gray-800 bg-transparent px-px py-2 text-2xs placeholder:italic focus:border-b focus:outline-none focus:ring-gray-200 ${
              errors.password ? 'border-pink-500/60' : ''
            }`}
          />
          {errors.password && <ErrorMessage message={errors.password} />}
        </div>
        <div className='my-2 w-full'>
          <label
            htmlFor='password'
            className='text-3xs uppercase text-gray-500'
          >
            Confirm Password
          </label>
          <input
            required
            name='confirmPassword'
            type='password'
            placeholder='confirm your password'
            onChange={e => {
              setConfirmPassword(e.target.value)
              setErrors({})
              setError('')
            }}
            className={`mb-1 w-full border-b border-gray-800 bg-transparent px-px py-2 text-2xs placeholder:italic focus:border-b focus:outline-none focus:ring-gray-200 ${
              errors.confirmPassword ? 'border-pink-500/60' : ''
            }`}
          />
          {errors.confirmPassword && (
            <ErrorMessage message={errors.confirmPassword} />
          )}
        </div>
        <button
          disabled={!password || !confirmPassword}
          className='my-2 rounded-lg border border-violet-300 px-4 py-3 text-2xs text-violet-100 hover:border-violet-100 disabled:pointer-events-none disabled:opacity-30'
        >
          Finish
        </button>
        {error && <ErrorMessage message={error} />}
      </form>
    </>
  )
}

Finish.layout = page => <Login>{page}</Login>
