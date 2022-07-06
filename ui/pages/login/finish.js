import { useRouter } from 'next/router'
import { useState } from 'react'
import { useSWRConfig } from 'swr'

import Login from '../../components/layouts/login'

export default function Finish() {
  const router = useRouter()

  const [password, setPassword] = useState('')
  const [error, setError] = useState('')
  const { mutate } = useSWRConfig()

  const { query } = router
  const { user } = query

  if (!user) {
    router.replace('/login')
  }

  async function finish(e) {
    e.preventDefault()

    try {
      const res = await fetch(`/api/users/${user}`, {
        method: 'PUT',
        body: JSON.stringify({ password }),
      })

      if (!res.ok) {
        throw await res.json()
      }

      await mutate('/api/users/self')

      router.replace('/')
    } catch (e) {
      setError(e.message || 'Invalid password')
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
        <div className='my-4 w-full'>
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
              setError('')
            }}
            className={`w-full border-b border-gray-800 bg-transparent px-px py-3 text-2xs placeholder:italic focus:border-b focus:outline-none focus:ring-gray-200 ${
              error ? 'border-pink-500/60' : ''
            }`}
          />
        </div>
        <button
          disabled={!password}
          className='my-2 rounded-lg border border-violet-300 px-4 py-3 text-2xs text-violet-100 hover:border-violet-100 disabled:pointer-events-none disabled:opacity-30'
        >
          Finish
        </button>
        {error && (
          <p className='absolute -bottom-3.5 mx-auto w-full text-center text-2xs text-pink-400'>
            {error}
          </p>
        )}
      </form>
    </>
  )
}

Finish.layout = page => <Login>{page}</Login>
