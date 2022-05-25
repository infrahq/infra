import { useRouter } from 'next/router'
import { useState } from 'react'
import { useSWRConfig } from 'swr'

import Login from '../../components/layouts/login'

export default function Finish () {
  const router = useRouter()

  const [password, setPassword] = useState('')
  const [error, setError] = useState('')
  const { mutate } = useSWRConfig()

  const { query } = router
  const { user } = query

  if (!user) {
    router.replace('/login')
  }

  async function finish (e) {
    e.preventDefault()

    try {
      await fetch(`/api/users/${user}`, { method: 'PUT', body: JSON.stringify({ password }) })
      await mutate('/api/users/self')
      router.replace('/')
    } catch (e) {
      setError(e.message || 'invalid password')
      setError('Invalid password')
    }

    return false
  }

  return (
    <>
      <h1 className='text-base leading-snug font-bold'>Login to Infra</h1>
      <h2 className='text-xs text-center max-w-[260px] my-3 text-gray-300'>You've used a one time password.<br />Set your new password to continue.</h2>

      <form onSubmit={finish} className='flex flex-col w-full max-w-sm relative'>
        <div className='w-full my-4'>
          <label htmlFor='password' className='text-3xs text-gray-500 uppercase'>New Password</label>
          <input
            required
            name='password'
            type='password'
            placeholder='enter your new password'
            onChange={e => {
              setPassword(e.target.value)
              setError('')
            }}
            className={`w-full bg-transparent border-b border-gray-800 text-2xs px-px py-3 focus:outline-none focus:border-b focus:ring-gray-200 placeholder:italic ${error ? 'border-pink-500/60' : ''}`}
          />
        </div>
        <button disabled={!password} className='border border-violet-300 hover:border-violet-100 my-2 text-2xs px-4 py-3 rounded-lg disabled:pointer-events-none text-violet-100 disabled:opacity-30'>
          Finish
        </button>
        {error && <p className='absolute -bottom-3.5 w-full mx-auto text-2xs text-pink-400 text-center'>{error}</p>}
      </form>
    </>
  )
}

Finish.layout = page => <Login>{page}</Login>
