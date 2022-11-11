import { useState } from 'react'

import { EnvelopeOpenIcon } from '@heroicons/react/24/outline'

import LoginLayout from '../../components/layouts/login'

export default function ForgotDomain() {
  const [email, setEmail] = useState('')
  const [error, setError] = useState('')
  const [submitted, setSubmitted] = useState(false)

  async function onSubmit(e) {
    e.preventDefault()

    setSubmitted(true)
    setError('')

    try {
      const res = await fetch('/api/forgot-domain-request', {
        method: 'post',
        body: JSON.stringify({
          email,
        }),
      })

      await jsonBody(res)
    } catch (e) {
      setSubmitted(false)
      setError(e.message)
    }

    return false
  }

  return (
    <div className='flex w-full flex-col items-center px-10 pt-4 pb-6'>
      <h1 className='text-base font-bold leading-snug'>
        Find your organization
      </h1>
      <h2 className='my-1.5 max-w-md text-center text-xs text-gray-500'>
        Enter your email, and we&apos;ll send you a list of organizations you
        are a member of.
      </h2>

      {submitted ? (
        <p className='my-3 flex max-w-[260px] flex-1 flex-col items-center justify-center text-center text-xs text-gray-600'>
          <EnvelopeOpenIcon className='mb-2 h-10 w-10 stroke-1 text-gray-400' />
          Please check your inbox. We&apos;ve sent you an email with a list of
          your organizations.
        </p>
      ) : (
        <form
          onSubmit={onSubmit}
          className='relative flex w-full max-w-sm flex-1 flex-col justify-center'
        >
          <div className='my-2'>
            <label
              htmlFor='name'
              className='text-2xs font-medium text-gray-700'
            >
              Email
            </label>
            <input
              required
              autoFocus
              name='name'
              type='email'
              onChange={e => {
                setEmail(e.target.value)
                setError('')
              }}
              className={`mt-1 block w-full rounded-md shadow-sm focus:border-blue-500 focus:ring-blue-500 sm:text-sm ${
                error ? 'border-red-500' : 'border-gray-300'
              }`}
            />
          </div>
          <button
            disabled={!email}
            type='submit'
            className='mt-4 mb-2 flex w-full cursor-pointer justify-center rounded-md border border-transparent bg-blue-500 py-2 px-4 text-sm font-medium text-white shadow-sm hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-30'
          >
            Find your organization
          </button>
          {error && <p className='my-1 text-xs text-red-500'>{error}</p>}
        </form>
      )}
    </div>
  )
}

ForgotDomain.layout = page => <LoginLayout>{page}</LoginLayout>
