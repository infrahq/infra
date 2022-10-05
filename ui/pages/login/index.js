import { useRouter } from 'next/router'
import { useState } from 'react'
import useSWR from 'swr'
import Cookies from 'universal-cookie'
import Link from 'next/link'

import { useUser } from '../../lib/hooks'
import { providers as providersList } from '../../lib/providers'
import { useServerConfig } from '../../lib/serverconfig'

import LoginLayout from '../../components/layouts/login'

function oidcLogin({ id, clientID, authURL, scopes }, next) {
  window.localStorage.setItem('providerID', id)
  if (next) {
    window.localStorage.setItem('next', next)
  }

  const state = [...Array(10)]
    .map(() => (~~(Math.random() * 36)).toString(36))
    .join('')
  window.localStorage.setItem('state', state)

  const redirectURL = window.location.origin + '/login/callback'
  window.localStorage.setItem('redirectURL', redirectURL)

  document.location.href = `${authURL}?redirect_uri=${redirectURL}&client_id=${clientID}&response_type=code&scope=${scopes.join(
    '+'
  )}&state=${state}`
}

export function saveToVisitedOrgs(domain, baseDomain, orgName) {
  const cookies = new Cookies()

  let visitedOrgs = cookies.get('orgs') || []

  if (!visitedOrgs.find(x => x.url === domain)) {
    visitedOrgs.push({
      url: domain,
      name: orgName,
    })

    cookies.set('orgs', visitedOrgs, {
      path: '/',
      domain: `.${baseDomain}`,
    })
  }
}

export function Providers({ providers }) {
  const router = useRouter()
  const { next } = router.query
  return (
    <>
      <div className='mt-4 w-full text-sm'>
        {providers.map(
          p =>
            p.kind && (
              <button
                onClick={() => oidcLogin({ ...p }, next)}
                key={p.id}
                title={`${p.name} — ${p.url}`}
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
                      <span>Log in with </span>
                      <span className='capitalize'>{p.name}</span>
                    </div>
                  ) : (
                    'Single Sign-On'
                  )}
                </span>
              </button>
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
  const { baseDomain, isEmailConfigured } = useServerConfig()
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
        router.replace({
          pathname: '/login/finish',
          query: next ? { user: data.userID, next } : { user: data.userID },
        })

        return false
      }

      router.replace(next ? decodeURIComponent(next) : '/')

      saveToVisitedOrgs(
        window.location.host,
        baseDomain,
        data?.organizationName
      )
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
      <h2 className='my-2 text-center text-sm text-gray-500'>
        Welcome back to Infra
      </h2>
      {providers?.length > 0 && (
        <>
          <Providers providers={providers || []} />
          <div className='relative mt-6 mb-2 w-full'>
            <div
              className='absolute inset-0 flex items-center'
              aria-hidden='true'
            >
              <div className='w-full border-t border-gray-200' />
            </div>
            <div className='relative flex justify-center text-sm'>
              <span className='bg-white px-2 text-2xs text-gray-400'>OR</span>
            </div>
          </div>
        </>
      )}
      <form onSubmit={onSubmit} className='relative flex w-full flex-col'>
        <div className='my-2 w-full'>
          <label htmlFor='name' className='text-2xs font-medium text-gray-700'>
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
            className='rounded-mdshadow-sm mt-1 block w-full focus:border-blue-500 focus:ring-blue-500 sm:text-sm'
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
    </div>
  )
}

Login.layout = page => <LoginLayout>{page}</LoginLayout>
