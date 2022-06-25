import useSWR from 'swr'
import { useState } from 'react'
import { useRouter } from 'next/router'
import Head from 'next/head'
import Link from 'next/link'

import Fullscreen from '../../components/layouts/fullscreen'
import ErrorMessage from '../../components/error-message'

export default function PasswordReset() {
  const router = useRouter()

  const { data: auth } = useSWR('/api/users/self')

  const [password, setPassword] = useState('')
  const [confirmPassword, setConfirmPassword] = useState('')
  const [error, setError] = useState('')

  async function onSubmit(e) {
    e.preventDefault()

    if (password !== confirmPassword) {
      setError('password does not match')
      return false
    }

    setError('')

    try {
      const rest = await fetch(`/api/users/${auth?.id}`, {
        method: 'PUT',
        body: JSON.stringify({
          ...auth,
          password: confirmPassword,
        }),
      })

      const data = await rest.json()

      if (!rest.ok) {
        throw data
      }

      router.replace('/settings?resetPassword=success')
    } catch (e) {
      setError(e.message || 'something went wrong, please try again')
    }
  }

  return (
    <div className='px-3 pt-8 pb-3'>
      <Head>
        <title>Password Reset</title>
      </Head>
      <div className='mx-auto flex w-full max-w-xs flex-col items-center justify-center'>
        <div className='mb-4 rounded-full border border-violet-200/25 p-2.5'>
          <img alt='infra icon' className='h-12 w-12' src='/infra-color.svg' />
        </div>
        <h1 className='text-base font-bold leading-snug'>Reset Password</h1>
      </div>
      <form onSubmit={onSubmit} className='mt-12 flex flex-col'>
        <div className='my-2 w-full'>
          <label htmlFor='name' className='text-3xs uppercase text-gray-500'>
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
            className={`w-full border-b border-gray-800 bg-transparent px-px py-2 text-2xs placeholder:italic focus:border-b focus:border-gray-200 focus:outline-none ${
              error ? 'border-pink-500/60' : ''
            }`}
          />
        </div>
        <div className='my-2 w-full'>
          <label
            htmlFor='password'
            className='text-3xs uppercase text-gray-500'
          >
            Confirm New Password
          </label>
          <input
            required
            name='confirmPassword'
            type='password'
            placeholder='confirm your new password'
            onChange={e => {
              setConfirmPassword(e.target.value)
              setError('')
            }}
            className={`w-full border-b border-gray-800 bg-transparent px-px py-2 text-2xs placeholder:italic focus:border-b focus:outline-none focus:ring-gray-200 ${
              error ? 'border-pink-500/60' : ''
            }`}
          />
        </div>
        <div className='mt-6 flex flex-row items-center justify-end'>
          <Link href='/settings'>
            <a className='border-0 px-6 py-3 text-2xs uppercase text-gray-400 hover:text-white focus:text-white focus:outline-none'>
              Cancel
            </a>
          </Link>
          <button
            type='submit'
            disabled={!password || !confirmPassword}
            className='rounded-md border border-violet-300 px-5 py-2.5 text-center text-2xs text-violet-100 disabled:opacity-30'
          >
            Reset
          </button>
        </div>
        {error && <ErrorMessage message={error} center />}
      </form>
    </div>
  )
}

PasswordReset.layout = page => (
  <Fullscreen closeHref='/settings'>{page}</Fullscreen>
)
