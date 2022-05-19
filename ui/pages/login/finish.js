import { useRouter } from 'next/router'
import { useState } from 'react'
import { useSWRConfig } from 'swr'

export default function () {
  const router = useRouter()

  const [password, setPassword] = useState('')
  const [error, setError] = useState('')
  const { mutate } = useSWRConfig()

  const { query } = router
  const { id } = query

  if (!id) {
    router.replace('/login')
  }

  async function finish () {
    try {
      await fetch(`/v1/identities/${id}`, { method: 'PUT', body: JSON.stringify({ password }) })
      await mutate('/v1/identities/self')
      router.replace('/')
    } catch (e) {
      setError(e.message || 'invalid password')
      setError('Invalid password')
    }
  }

  return (
    <div className='flex flex-col justify-center items-center h-full w-full max-w-sm mx-auto mb-48'>
      <img className='text-white w-10 h-10' src='/infra-icon.svg' />
      <h1 className='my-5 text-3xl font-light tracking-tight'>Login to Infra</h1>
      <h2 className='text-center mt-4 text-gray-300'>You've used a one time password.<br />Set your password to continue.</h2>
      <form
        className='w-full flex flex-col max-w-sm my-6 relative'
        onSubmit={e => {
          e.preventDefault()
          finish()
        }}
      >
        <input
          required
          type='password'
          name='password'
          id='password'
          className={`block w-full px-4 py-2 text-md -my-px border font-light rounded-lg text-gray-100 bg-gray-900/50 placeholder-gray-500 ${error ? 'border-red-500' : 'border-gray-800'}`}
          placeholder='password'
          onChange={e => {
            setError('')
            setPassword(e.target.value)
          }}
        />
        <input type='submit' value='Login' className='w-full my-3 bg-gray-500/20 hover:bg-gray-500/25 py-2.5 rounded-md text-white text-md hover:cursor-pointer' />
        {error && (
          <p className='mt-2 text-sm absolute text-red-500 -bottom-4'>
            {error}
          </p>
        )}
      </form>
    </div>
  )
}
