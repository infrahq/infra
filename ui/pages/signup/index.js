import { useState } from 'react'
import { useSWRConfig } from 'swr'
import { useRouter } from 'next/router'

import HeaderIcon from '../../components/header-icon'
import ErrorMessage from '../../components/error-message'

export default function () {
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

      mutate('/v1/users/self', { optimisticData: { name: email } })
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
    <div className='h-auto w-full max-w-sm mx-auto overflow-hidden'>
      <div className='flex flex-col justify-center items-center px-5 py-5 mt-40 border rounded-lg border-gray-950'>
        <HeaderIcon size={12} iconPath='/infra-color.svg' />
        <h1 className='text-header font-bold'>Welcome to Infra</h1>
        <h2 className='text-title text-center max-w-md my-3 text-gray-300'>You've successfully installed Infra.<br />Set up your admin user to get started.</h2>
        <form onSubmit={onSubmit} className='flex flex-col w-full max-w-sm my-8'>
          <div className='w-full my-4'>
            <div className="text-label text-gray-200 uppercase">Email</div>
            <input 
              autoFocus 
              type='email' 
              placeholder='email@address.com' 
              onChange={e => setEmail(e.target.value)} 
              className={`w-full bg-transparent border-b border-gray-950 text-name px-px mt-2 py-3 focus:outline-none focus:border-b focus:border-gray-200 placeholder:italic ${errors.email ? 'border-pink-300' : ''}`} />
            {errors.email && <ErrorMessage message={errors.email} />}
          </div>
          <div className='w-full my-4'>
            <div className="text-label text-gray-200 uppercase">Password</div>
            <input 
              type='password' 
              placeholder='enter your password' 
              onChange={e => setPassword(e.target.value)} 
              className={`w-full bg-transparent border-b border-gray-950 text-name px-px mt-2 py-3 focus:outline-none focus:border-b focus:border-gray-200 placeholder:italic ${errors.password ? 'border-pink-300' : ''}`} />
            {errors.password && <ErrorMessage message={errors.password} />}
          </div>

          <button disabled={!email || !password} className='bg-gradient-to-tr mt-5 from-indigo-300 to-pink-100 hover:from-indigo-200 hover:to-pink-50 rounded-md p-0.5 my-2 disabled:opacity-30'>
            <div className='bg-black text-purple-50 rounded-md text-name px-4 py-3'>
              Get Started
            </div>
            {error && <ErrorMessage message={error} center />}
          </button>
        </form>
      </div>
    </div>
  )
}
