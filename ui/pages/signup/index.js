import { useState } from 'react'
import { useSWRConfig } from 'swr'
import { useRouter } from 'next/router'

import Login from '../../components/layouts/login'
import ErrorMessage from '../../components/error-message'

export default function Signup () {
  const { mutate } = useSWRConfig()
  const router = useRouter()

  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState('')
  const [errors, setErrors] = useState({})

  async function onSubmit (e) {
    e.preventDefault()

    setErrors({})
    setError('')

    try {
      // signup
      let res = await fetch('/v1/signup', {
        method: 'POST',
        body: JSON.stringify({
          email,
          password
        })
      })

      if (!res.ok) {
        throw await res.json()
      }

      // login
      res = await fetch('/v1/login', {
        method: 'POST',
        body: JSON.stringify({
          passwordCredentials: {
            email,
            password
          }
        })
      })

      if (!res.ok) {
        throw await res.json()
      }

      mutate('/v1/identities/self', { optimisticData: { name: email } })
      mutate('/v1/signup', { optimisticData: { enabled: false } })
      router.replace('/')
    } catch (e) {
      if (e.fieldErrors) {
        const errors = {}
        for (const error of e.fieldErrors) {
          errors[error.fieldName.toLowerCase()] = error.errors[0] || 'invalid value'
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
      <h1 className='text-base leading-snug font-bold'>Welcome to Infra</h1>
      <h2 className='text-[13px] text-center max-w-md my-1.5 text-gray-400'>You've successfully installed Infra.<br />Set up your admin user to get started.</h2>
      <form onSubmit={onSubmit} className='flex flex-col w-full max-w-sm'>
        <div className='w-full my-4'>
          <label htmlFor='email' className='text-xxs text-gray-400 uppercase'>Email</label>
          <input
            autoFocus
            name='email'
            type='email'
            placeholder='email@address.com'
            onChange={e => setEmail(e.target.value)}
            className={`w-full bg-transparent border-b border-gray-800 text-xs px-px mt-2 py-3 focus:outline-none focus:border-b focus:border-gray-200 placeholder:italic ${errors.email ? 'border-pink-500/60' : ''}`}
          />
          {errors.email && <ErrorMessage message={errors.email} />}
        </div>
        <div className='w-full my-4'>
          <label htmlFor='password' className='text-xxs text-gray-400 uppercase'>Password</label>
          <input
            type='password'
            placeholder='enter your password'
            onChange={e => setPassword(e.target.value)}
            className={`w-full bg-transparent border-b border-gray-800 text-xs px-px mt-2 py-3 focus:outline-none focus:border-b focus:border-gray-200 placeholder:italic ${errors.password ? 'border-pink-500/60' : ''}`}
          />
          {errors.password && <ErrorMessage message={errors.password} />}
        </div>

        <button disabled={!email || !password} className='border border-violet-300 hover:border-violet-100 my-2 text-xs px-4 py-3 rounded-lg disabled:pointer-events-none text-violet-100 disabled:opacity-30'>
          Get Started
          {error && <ErrorMessage message={error} center />}
        </button>
      </form>
    </>
  )
}

Signup.layout = page => <Login>{page}</Login>
