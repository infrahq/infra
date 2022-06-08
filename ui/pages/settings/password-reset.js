import useSWR from 'swr'
import { useState } from 'react'
import { useRouter } from 'next/router'
import Head from 'next/head'
import Link from 'next/link'

import Fullscreen from '../../components/layouts/fullscreen'
import ErrorMessage from '../../components/error-message'

export default function PasswordReset () {
  const router = useRouter()

  const { data: auth } = useSWR('/api/users/self')

  const [password, setPassword] = useState('')
  const [confirmPassword, setConfirmPassword] = useState('')
  const [error, setError] = useState('')

  async function onSubmit (e) {
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
          id: auth?.id,
          password: confirmPassword
        })
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
    <div className='pt-8 px-3 pb-3'>
      <Head>
        <title>Password Reset</title>
      </Head>
      <div className='flex flex-col w-full max-w-xs mx-auto justify-center items-center'>
        <div className='border border-violet-200/25 rounded-full p-2.5 mb-4'>
          <img className='w-12 h-12' src='/infra-color.svg' />
        </div>
        <h1 className='text-base leading-snug font-bold'>Reset Password</h1>
      </div>
      <form onSubmit={onSubmit} className='flex flex-col mt-12'>
        <div className='w-full my-2'>
          <label htmlFor='name' className='text-3xs text-gray-500 uppercase'>New Password</label>
          <input
            required
            name='password'
            type='password'
            placeholder='enter your new password'
            onChange={e => {
              setPassword(e.target.value)
              setError('')
            }}
            className={`w-full bg-transparent border-b border-gray-800 text-2xs px-px py-2 focus:outline-none focus:border-b focus:border-gray-200 placeholder:italic ${error ? 'border-pink-500/60' : ''}`}
          />
        </div>
        <div className='w-full my-2'>
          <label htmlFor='password' className='text-3xs text-gray-500 uppercase'>Confirm New Password</label>
          <input
            required
            name='confirmPassword'
            type='password'
            placeholder='confirm your new password'
            onChange={e => {
              setConfirmPassword(e.target.value)
              setError('')
            }}
            className={`w-full bg-transparent border-b border-gray-800 text-2xs px-px py-2 focus:outline-none focus:border-b focus:ring-gray-200 placeholder:italic ${error ? 'border-pink-500/60' : ''}`}
          />
        </div>
        <div className='flex flex-row justify-end mt-6 items-center'>
          <Link href='/settings'>
            <a className='uppercase border-0 hover:text-white px-6 py-3 focus:outline-none focus:text-white text-gray-400 text-2xs'>Cancel</a>
          </Link>
          <button
            type='submit'
            disabled={!password || !confirmPassword}
            className='border border-violet-300 text-2xs text-violet-100 rounded-md px-5 py-2.5 text-center disabled:opacity-30'
          >
            Reset
          </button>
        </div>
        {error && <ErrorMessage message={error} center />}
      </form>
    </div>
  )
}

PasswordReset.layout = page => <Fullscreen closeHref='/settings'>{page}</Fullscreen>
