import { useRouter } from 'next/router'
import { useState } from 'react'
import useSWR from 'swr'
import Cookies from 'universal-cookie'
import Link from 'next/link'

import { useUser } from '../../lib/hooks'
import { useServerConfig } from '../../lib/serverconfig'

import LoginLayout from '../../components/layouts/login'
import UpdatePassword from '../../components/update-password'
import Providers from '../../components/providers'

export function saveToVisitedOrgs(domain, orgName) {
  const cookies = new Cookies()

  let visitedOrgs = cookies.get('orgs') || []

  if (!visitedOrgs.find(x => x.url === domain)) {
    visitedOrgs.push({
      url: domain,
      name: orgName,
    })

    // set the cookie domain to a general base domain
    let cookieDomain = window.location.host
    let parts = cookieDomain.split('.')
    if (parts.length > 2) {
      parts.shift() // remove the org
      cookieDomain = parts.join('.') // join the last two parts of the domain
    }

    cookies.set('orgs', visitedOrgs, {
      path: '/',
      domain: `.${cookieDomain}`,
    })
  }
}

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
  const { isEmailConfigured } = useServerConfig()
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

      router.replace(next ? decodeURIComponent(next) : '/')

      saveToVisitedOrgs(window.location.host, data?.organizationName)
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
                buttonPrompt={'Log in with'}
                callbackPath={'/login/callback'}
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
            <div className='my-2 w-full'>
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
            </div>
            <div className='my-2 w-full'>
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
                data-testid='form-field-password'
                onChange={e => {
                  setPassword(e.target.value)
                  setErrors({})
                  setError('')
                }}
                className='mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-blue-500 focus:ring-blue-500 sm:text-sm'
              />
            </div>
            {isEmailConfigured && (
              <div className='mt-4 flex items-center justify-end text-sm'>
                <Link href='/password-reset'>
                  <a className='font-medium text-blue-600 hover:text-blue-500'>
                    Forgot your password?
                  </a>
                </Link>
              </div>
            )}
            <button className='mt-4 mb-2 flex w-full cursor-pointer justify-center rounded-md border border-transparent bg-blue-500 py-2 px-4 text-sm font-medium text-white shadow-sm hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2'>
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
