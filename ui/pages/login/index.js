import { useRouter } from 'next/router'
import { useState } from 'react'
import useSWR from 'swr'
import Link from 'next/link'
import Tippy from '@tippyjs/react'
import Cookies from 'universal-cookie'

import { useUser } from '../../lib/hooks'
import { providers as providersList } from '../../lib/providers'
import { useServerConfig } from '../../lib/serverconfig'
import { saveToVisitedOrgs, currentBaseDomain } from '../../lib/login'

import LoginLayout from '../../components/layouts/login'
import UpdatePassword from '../../components/update-password'

function oidcLogin(
  { baseDomain, loginDomain, id, clientID, authURL, scopes },
  next
) {
  window.localStorage.setItem('providerID', id)
  if (next) {
    window.localStorage.setItem('next', next)
  }

  const state = [...Array(10)]
    .map(() => (~~(Math.random() * 36)).toString(36))
    .join('')
  window.localStorage.setItem('state', state)

  if (baseDomain === '') {
    // this is possible if not configured on the server
    // fallback to the browser domain
    baseDomain = currentBaseDomain()
  }

  let redirectURL = window.location.origin + '/login/callback'
  if (id === '') {
    // managed oidc providers (social login) need to be sent to the base redirect URL before they are redirected to org login
    const cookies = new Cookies()
    cookies.set('finishLogin', window.location.host, {
      path: '/',
      domain: `.${baseDomain}`,
      sameSite: 'lax',
    })
    redirectURL = window.location.protocol + '//' + loginDomain + '/redirect' // go to the social login redirect specified by the server
  }
  window.localStorage.setItem('redirectURL', redirectURL)

  document.location.href = `${authURL}?redirect_uri=${redirectURL}&client_id=${clientID}&response_type=code&scope=${scopes.join(
    '+'
  )}&state=${state}`
}

function Providers({ baseDomain, loginDomain, providers }) {
  const router = useRouter()
  const { next } = router.query
  return (
    <>
      <div className='mt-4 w-full text-sm'>
        {providers.map(
          p =>
            p.kind && (
              <div key={p.id}>
                <Tippy
                  content={`${p.name} — ${p.url}`}
                  className='whitespace-no-wrap z-8 relative w-auto rounded-md bg-black p-2 text-xs text-white shadow-lg'
                  interactive={true}
                  interactiveBorder={20}
                  offset={[0, 5]}
                  delay={[250, 0]}
                  placement='top'
                >
                  <button
                    onClick={() =>
                      oidcLogin({ baseDomain, loginDomain, ...p }, next)
                    }
                    className='my-2 inline-flex w-full items-center rounded-md border border-gray-300 bg-white py-2.5 px-4 text-gray-500 shadow-sm hover:bg-gray-100 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2'
                  >
                    <img
                      alt='identity provider icon'
                      className='h-4'
                      src={`/providers/${p.kind}.svg`}
                    />
                    <span className='items-center truncate pl-4 text-gray-800'>
                      {providersList.filter(i => i.kind === p.kind) ? (
                        <div className='truncate'>
                          <span>Log in with {p.name}</span>
                        </div>
                      ) : (
                        'Single Sign-On'
                      )}
                    </span>
                  </button>
                </Tippy>
              </div>
            )
        )}
      </div>
    </>
  )
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
                baseDomain={baseDomain}
                loginDomain={loginDomain}
                providers={providers || []}
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
