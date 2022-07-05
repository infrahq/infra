import { useState } from 'react'
import { useSWRConfig } from 'swr'
import { useRouter } from 'next/router'

import Login from '../../components/layouts/login'
import ErrorMessage from '../../components/error-message'

export default function Signup() {
  const { mutate } = useSWRConfig()
  const router = useRouter()

  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState('')
  const [errors, setErrors] = useState({})

  async function onSubmit(e) {
    e.preventDefault()

    setErrors({})
    setError('')

    try {
      // signup
      let res = await fetch('/api/signup', {
        method: 'POST',
        body: JSON.stringify({
          email,
          password,
        }),
      })

      if (!res.ok) {
        throw await res.json()
      }

      // login
      res = await fetch('/api/login', {
        method: 'POST',
        body: JSON.stringify({
          passwordCredentials: {
            email,
            password,
          },
        }),
      })

      if (!res.ok) {
        throw await res.json()
      }

      await mutate('/api/signup')
      await mutate('/api/users/self')

      router.replace('/')
    } catch (e) {
      if (e.fieldErrors) {
        const errors = {}
        for (const error of e.fieldErrors) {
          errors[error.fieldName.toLowerCase()] =
            error.errors[0] || 'invalid value'
        }
        setErrors(errors)
      } else {
        setError(e.message)
      }
    }

    return false
  }

  return (
    <>
      <h1 className='text-base font-bold leading-snug'>Welcome to Infra</h1>
      <h2 className='my-1.5 max-w-md text-center text-xs text-gray-400'>
        You&apos;ve successfully installed Infra.
        <br />
        Set up your admin user to get started.
      </h2>
      <form onSubmit={onSubmit} className='flex w-full max-w-sm flex-col'>
        <div className='my-4 w-full'>
          <label htmlFor='email' className='text-3xs uppercase text-gray-400'>
            Email
          </label>
          <input
            autoFocus
            name='email'
            type='email'
            placeholder='email@address.com'
            onChange={e => setEmail(e.target.value)}
            className={`mt-2 w-full border-b border-gray-800 bg-transparent px-px py-3 text-2xs placeholder:italic focus:border-b focus:border-gray-200 focus:outline-none ${
              errors.email ? 'border-pink-500/60' : ''
            }`}
          />
          {errors.email && <ErrorMessage message={errors.email} />}
        </div>
        <div className='my-4 w-full'>
          <label
            htmlFor='password'
            className='text-3xs uppercase text-gray-400'
          >
            Password
          </label>
          <input
            type='password'
            placeholder='enter your password'
            onChange={e => setPassword(e.target.value)}
            className={`mt-2 w-full border-b border-gray-800 bg-transparent px-px py-3 text-2xs placeholder:italic focus:border-b focus:border-gray-200 focus:outline-none ${
              errors.password ? 'border-pink-500/60' : ''
            }`}
          />
          {errors.password && <ErrorMessage message={errors.password} />}
        </div>

        <button
          disabled={!email || !password}
          className='my-2 rounded-lg border border-violet-300 px-4 py-3 text-2xs text-violet-100 hover:border-violet-100 disabled:pointer-events-none disabled:opacity-30'
        >
          Get Started
          {error && <ErrorMessage message={error} center />}
        </button>
      </form>
    </>
  )
}

Signup.layout = page => <Login>{page}</Login>
