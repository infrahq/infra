import { useState } from 'react'
import Link from 'next/link'
import { useSWRConfig } from 'swr'
import { useRouter } from 'next/router'

import HeaderIcon from '../../components/dashboard/headerIcon'

export default function () {
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [errors, setErrors] = useState({})
  const [error, setError] = useState('')
  const { mutate } = useSWRConfig()
  const router = useRouter()

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
      console.log(email, password)
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

      mutate('/v1/introspect', { optimisticData: { name: email } })
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
    <div className='flex flex-col justify-center items-center h-full w-full max-w-md mx-auto mb-48'>
      <HeaderIcon width={12} iconPath='/infra-color.svg' />
      <h1 className='mt-5 text-md font-bold'>Welcome to Infra</h1>
      <h2 className='text-sm text-center max-w-xs my-2 text-gray-400'>You've successfully installed Infra.<br />Set up your admin user to get started.</h2>
      <form onSubmit={onSubmit} className='flex flex-col w-full max-w-sm my-8'>
        <input autoFocus type="email" placeholder='Email' onChange={e => setEmail(e.target.value)} className={`bg-purple-100/5 border border-zinc-800 text-sm px-5 mt-2 py-3 rounded-full focus:outline-none focus:ring focus:ring-cyan-600 ${errors.email ? 'border-pink-500' : ''}`} />
        {errors.email && <p className='px-4 mb-1 text-sm text-pink-500'>{errors.email}</p>}
        <input type='password' placeholder='Password' onChange={e => setPassword(e.target.value)} className={`bg-purple-100/5 border border-zinc-800 text-sm px-5 mt-2 py-3 rounded-full focus:outline-none focus:ring focus:ring-cyan-600 ${errors.password ? 'border-pink-500' : ''}`} />
        {errors.password && <p className='px-4 mb-1 text-sm text-pink-500'>{errors.password}</p>}
        <button className='bg-gradient-to-tr mt-5 from-indigo-300 to-pink-100 hover:from-indigo-200 hover:to-pink-50 rounded-full p-0.5 my-2'>
          <div className='bg-black rounded-full text-sm px-4 py-3'>
            Get Started
          </div>
          {error && <p className='mt-2 text-sm text-pink-500 text-center'>{error}</p>}
        </button>
      </form>
    </div>
  )
}
