import { useRouter } from 'next/router'
import { useState } from 'react'
import useSWR, { useSWRConfig } from 'swr'


import LoginLayout from '../../components/layouts/login'

export default function PasswordReset() {
  const { mutate } = useSWRConfig()
  const router = useRouter()
  const { token } = router.query

  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState('')
  const [submitted, setSubmitted] = useState(false)

  async function onSubmitSetPassword(e) {
    e.preventDefault()

    try {
      const res = await fetch('/api/password-reset', {
        method: 'post',
        body: JSON.stringify({
          token,
          password,
        }),
      })

      if (!res.ok) {
        throw await res.json()
      }

      const data = await res.json()

      await mutate('/api/users/self')
      router.replace('/')
    } catch (e) {
      console.error(e)
    }

    return false
  }

  async function onSubmit(e) {
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

      const data = await res.json()
      setSubmitted(true)

    } catch (e) {
      console.error(e)
    }

    return false
  }

  return (
    <>
      {token ? (
        <>
          <h2 className='my-3 max-w-[260px] text-center text-xs text-gray-300'>
            Please set your password
          </h2>
          <div className='relative mt-4 w-full'>
            <div className='absolute inset-0 flex items-center' aria-hidden='true'>
              <div className='w-full border-t border-gray-800' />
            </div>
          </div>
          <form
            onSubmit={onSubmitSetPassword}
            className='relative flex w-full max-w-sm flex-col'
          >
            <div className='my-2 w-full'>
              <label
                htmlFor='password'
                className='text-3xs uppercase text-gray-500'
              >
                Password
              </label>
              <input
                required
                id='password'
                type='password'
                data-testid='form-field-password'
                placeholder='enter your password'
                onChange={e => {
                  setPassword(e.target.value)
                  setError('')
                }}
                className={`w-full border-b border-gray-800 bg-transparent px-px py-2 text-2xs placeholder:italic focus:border-b focus:outline-none focus:ring-gray-200 ${error ? 'border-pink-500/60' : ''
                  }`}
              />
            </div>
            <button
              disabled={!password}
              className='mt-6 mb-2 rounded-lg border border-violet-300 px-4 py-3 text-2xs text-violet-100 hover:border-violet-100 disabled:pointer-events-none disabled:opacity-30'
            >
              Set Password
            </button>
            {error && (
              <p className='absolute -bottom-3.5 mx-auto w-full text-center text-2xs text-pink-400'>
                {error}
              </p>
            )}
          </form>
        </>
      ) : (
        <>
          <h1 className='text-base font-bold leading-snug'>Password Reset</h1>
          {
            submitted ? (
              <p>Please check your email for the reset link</p>
            ) : (
              <>
                <h2 className='my-3 max-w-[260px] text-center text-xs text-gray-300'>
                  Please enter your email
                </h2>
                <div className='relative mt-4 w-full'>
                  <div className='absolute inset-0 flex items-center' aria-hidden='true'>
                    <div className='w-full border-t border-gray-800' />
                  </div>
                </div>
                <form
                  onSubmit={onSubmit}
                  className='relative flex w-full max-w-sm flex-col'
                >
                  <div className='my-2 w-full'>
                    <label htmlFor='email' className='text-3xs uppercase text-gray-500'>
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
                      className={`w-full border-b border-gray-800 bg-transparent px-px py-2 text-2xs placeholder:italic focus:border-b focus:border-gray-200 focus:outline-none ${error ? 'border-pink-500/60' : ''
                        }`}
                    />
                  </div>
                  <button
                    disabled={!email}
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
            )
          }

        </>
      )
      }
    </>
  )
}

PasswordReset.layout = page => <LoginLayout>{page}</LoginLayout>