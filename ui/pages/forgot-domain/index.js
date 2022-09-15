import { useState } from 'react'
import { MailIcon } from '@heroicons/react/outline'

import LoginLayout from '../../components/layouts/login'

export default function ForgotDomain() {
  const [email, setEmail] = useState('')
  const [error, setError] = useState('')
  const [submitted, setSubmitted] = useState(false)

  async function onSubmit(e) {
    e.preventDefault()
    setSubmitted(true)

    try {
      const res = await fetch('/api/forgot-domain-request', {
        method: 'post',
        body: JSON.stringify({
          email,
        }),
      })

      if (!res.ok) {
        throw await res.json()
      }

      await res.json()
    } catch (e) {
      console.error(e)
    }

    return false
  }

  return (
    <div className='flex min-h-[320px] w-full flex-col items-center px-10 py-8'>
      <h1 className='text-base font-bold leading-snug'>
        Find your organization
      </h1>
      <h2 className='my-1.5 mb-4 max-w-md text-center text-xs text-gray-500'>
        Choose an organization to log in to.
      </h2>
      {submitted ? (
        <p className='my-3 flex max-w-[260px] flex-1 flex-col items-center justify-center text-center text-xs text-gray-600'>
          <MailIcon className='mb-2 h-10 w-10 stroke-1 text-gray-400' />
          Please check your inbox. We&apos;ve sent you an email with a list of
          your organizations.
        </p>
      ) : (
        <>
          <h2 className='my-2 text-center text-sm text-gray-500'>
            Enter your email, and we&apos;ll send you a list of organizations
            you are a member of.
          </h2>
          <form
            onSubmit={onSubmit}
            className='relative flex w-full max-w-sm flex-1 flex-col justify-center'
          >
            <div className='my-2 w-full'>
              <label
                htmlFor='password'
                className='text-2xs font-medium text-gray-700'
              >
                Email
              </label>
              <input
                required
                autoFocus
                id='name'
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
            <button className='mt-4 mb-2 flex w-full cursor-pointer justify-center rounded-md border border-transparent bg-blue-500 py-2 px-4 font-medium text-white shadow-sm hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2 sm:text-sm'>
              Find your organization
            </button>
            {error && (
              <p className='absolute -bottom-3.5 mx-auto w-full text-center text-2xs text-red-500'>
                {error}
              </p>
            )}
          </form>
        </>
      )}
    </div>
  )
}

ForgotDomain.layout = page => <LoginLayout>{page}</LoginLayout>
