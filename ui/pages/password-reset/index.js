import { useRouter } from 'next/router'
import { useState } from 'react'
import { MailIcon } from '@heroicons/react/outline'

import LoginLayout from '../../components/layouts/login'
import PasswordResetForm from '../../components/password-reset-form'

export default function PasswordReset() {
  const router = useRouter()
  const { token } = router.query

  const [email, setEmail] = useState('')
  const [submitted, setSubmitted] = useState(false)

  async function onSubmit(e) {
    setSubmitted(true)
    e.preventDefault()

    try {
      const res = await fetch('/api/password-reset-request', {
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
    <div className='flex min-h-[280px] w-full flex-col items-center px-10 py-8'>
      {token ? (
        <>
          <h1 className='text-base font-bold leading-snug'>Reset password</h1>
          <h2 className='my-1.5 mb-4 max-w-md text-center text-xs text-gray-500'>
            Set your new password
          </h2>
          <PasswordResetForm />
        </>
      ) : (
        <>
          <h1 className='text-base font-bold leading-snug'>Password Reset</h1>
          {submitted ? (
            <p className='my-3 flex max-w-[260px] flex-1 flex-col items-center justify-center text-center text-xs text-gray-600'>
              <MailIcon className='mb-2 h-10 w-10 stroke-1 text-gray-400' />
              Please check your inbox. We&apos;ve sent you a link to reset your
              password
            </p>
          ) : (
            <>
              <h2 className='my-1.5 mb-4 max-w-md text-center text-xs text-gray-500'>
                Please enter your email to reset your password.
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
                    type='email'
                    onChange={e => setEmail(e.target.value)}
                    className='mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-blue-500 focus:ring-blue-500 sm:text-sm'
                  />
                </div>
                <button
                  disabled={submitted}
                  className='mt-4 mb-2 flex w-full cursor-pointer justify-center rounded-md border border-transparent bg-blue-500 py-2 px-4 font-medium text-white shadow-sm hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2 sm:text-sm'
                >
                  Reset Password
                </button>
              </form>
            </>
          )}
        </>
      )}
    </div>
  )
}

PasswordReset.layout = page => <LoginLayout>{page}</LoginLayout>
