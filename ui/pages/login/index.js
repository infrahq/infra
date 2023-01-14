import { useRouter } from 'next/router'
import { useState } from 'react'
import useSWR from 'swr'
import Link from 'next/link'

import { useUser } from '../../lib/hooks'
import { useServerConfig } from '../../lib/serverconfig'
import { saveToVisitedOrgs } from '../../lib/login'

import LoginLayout from '../../components/layouts/login'
import Providers, { oidcLogin } from '../../components/providers'
import UpdatePassword from '../../components/update-password'

export default function Login() {
  const { data: { items: providers } = {} } = useSWR(
    '/api/providers?limit=1000',
    {
      fallbackData: [],
    }
  )

  const router = useRouter()
  const { next } = router.query

  const [name, setName] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState('')
  const [errors, setErrors] = useState({})
  const [updatePasswordForUser, setUpdatePasswordForUser] = useState('')
  const { isEmailConfigured, baseDomain, loginDomain } = useServerConfig()
  const { login } = useUser()

  async function onSubmit(e) {
    e.preventDefault()

    try {
      const data = await login({
        passwordCredentials: {
          name,
          password,
        },
      })

      if (data.passwordUpdateRequired) {
        setUpdatePasswordForUser(data.userID)
        return false
      }

      saveToVisitedOrgs(window.location.host, data?.organizationName)
      router.replace(next ? decodeURIComponent(next) : '/')
    } catch (e) {
      console.error(e)
      if (e.fieldErrors) {
        const errors = {}
        for (const error of e.fieldErrors) {
          errors[error.fieldName.toLowerCase()] =
            error.errors[0] || 'invalid value'
        }
        setErrors(errors)
      } else {
        if (e.code === 401 && e.message === 'unauthorized') {
          setError('Invalid credentials')
        } else {
          setError(e.message)
        }
      }
    }

    return false
  }

  return (
    <div className='flex w-full flex-col items-center px-10 pt-4 pb-6'>
      <h1 className='mt-4 font-display text-2xl font-semibold leading-snug'>
        Log in
      </h1>
      {updatePasswordForUser !== '' ? (
        <UpdatePassword oldPassword={password} user={updatePasswordForUser} />
      ) : (
        <>
          <h2 className='my-2 text-center text-sm text-gray-500'>
            Welcome back to Infra
          </h2>
          {providers?.length > 0 && (
            <>
              <Providers
                providers={providers || []}
                baseDomain={baseDomain}
                loginDomain={loginDomain}
                authnFunc={oidcLogin}
                buttonPrompt={'Log in with'}
              />
              <div className='relative mt-6 mb-2 w-full'>
                <div
                  className='absolute inset-0 flex items-center'
                  aria-hidden='true'
                >
                  <div className='w-full border-t border-gray-200' />
                </div>
                <div className='relative flex justify-center text-sm'>
                  <span className='bg-white px-2 text-2xs text-gray-400'>
                    OR
                  </span>
                </div>
              </div>
            </>
          )}
          <form onSubmit={onSubmit} className='relative flex w-full flex-col'>
            <div className='space-y-2'>
              <>
                <label
                  htmlFor='name'
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
                    setName(e.target.value)
                    setErrors({})
                    setError('')
                  }}
                  className={`mt-1 block w-full rounded-md shadow-sm focus:border-blue-500 focus:ring-blue-500 sm:text-sm ${
                    errors.name ? 'border-red-500' : 'border-gray-300'
                  }`}
                />
                {errors.name && (
                  <p className='my-1 text-xs text-red-500'>{errors.name}</p>
                )}
              </>
              <>
                <label
                  htmlFor='password'
                  className='text-2xs font-medium text-gray-700'
                >
                  Password
                </label>
                <input
                  required
                  id='password'
                  type='password'
                  autoComplete='off'
                  onChange={e => {
                    setPassword(e.target.value)
                    setErrors({})
                    setError('')
                  }}
                  className='mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-blue-500 focus:ring-blue-500 sm:text-sm'
                />
              </>
            </div>
            {isEmailConfigured && (
              <div className='mt-4 flex items-center justify-end text-sm'>
                <Link
                  href='/password-reset'
                  className='font-medium text-blue-600 hover:text-blue-500'
                >
                  Forgot your password?
                </Link>
              </div>
            )}
            <button
              type='submit'
              className='mt-4 mb-2 flex w-full cursor-pointer justify-center rounded-md border border-transparent bg-blue-500 py-2 px-4 text-sm font-medium text-white shadow-sm hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2'
            >
              Log in
            </button>
            {error && <p className='my-1 text-xs text-red-500'>{error}</p>}
          </form>
        </>
      )}
    </div>
  )
}

Login.layout = page => <LoginLayout>{page}</LoginLayout>
