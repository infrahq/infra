import { useState } from 'react'

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
    <>
      <>
        <h1 className='text-base font-bold leading-snug'>Forgot Domain</h1>
        {submitted ? (
          <p className='my-3 max-w-[260px] text-xs text-gray-300'>
            Please check your email to find a list of organizations which match
            your email address.
          </p>
        ) : (
          <>
            <h2 className='my-3 max-w-[260px] text-center text-xs text-gray-300'>
              Please enter your email
            </h2>
            <div className='relative mt-4 w-full'>
              <div
                className='absolute inset-0 flex items-center'
                aria-hidden='true'
              >
                <div className='w-full border-t border-gray-800' />
              </div>
            </div>
            <form
              onSubmit={onSubmit}
              className='relative flex w-full max-w-sm flex-col'
            >
              <div className='my-2 w-full'>
                <label
                  htmlFor='email'
                  className='text-3xs uppercase text-gray-500'
                >
                  Email
                </label>
                <input
                  required
                  autoFocus
                  id='email'
                  placeholder='enter your email'
                  onChange={e => {
                    setEmail(e.target.value)
                    setError('')
                  }}
                  className={`w-full border-b border-gray-800 bg-transparent px-px py-2 text-2xs placeholder:italic focus:border-b focus:border-gray-200 focus:outline-none ${
                    error ? 'border-pink-500/60' : ''
                  }`}
                />
              </div>
              <button
                disabled={!email || submitted}
                className='mt-6 mb-2 rounded-lg border border-violet-300 px-4 py-3 text-2xs text-violet-100 hover:border-violet-100 disabled:pointer-events-none disabled:opacity-30'
              >
                Submit
              </button>
              {error && (
                <p className='absolute -bottom-3.5 mx-auto w-full text-center text-2xs text-pink-400'>
                  {error}
                </p>
              )}
            </form>
          </>
        )}
      </>
    </>
  )
}

ForgotDomain.layout = page => <LoginLayout>{page}</LoginLayout>
