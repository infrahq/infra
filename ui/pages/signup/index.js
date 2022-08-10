import { useState } from 'react'

import { useServerConfig } from '../../lib/serverconfig'
import Login from '../../components/layouts/login'

export default function Signup() {
  const [name, setName] = useState('')
  const [password, setPassword] = useState('')
  const [orgName, setOrgName] = useState('')
  const [subDomain, setSubDomain] = useState('')
  const [automaticOrgDomain, setAutomaticOrgDomain] = useState(true) // track if the user has manually specified the org domain
  const [submitted, setSubmitted] = useState(false)
  const [error, setError] = useState('')
  const [errors, setErrors] = useState({})

  const { baseDomain } = useServerConfig()

  async function onSubmit(e) {
    e.preventDefault()

    setSubmitted(true)

    try {
      let res = await fetch('/api/signup', {
        method: 'POST',
        body: JSON.stringify({
          name,
          password,
          org: {
            name: orgName,
            subDomain,
          },
        }),
      })

      if (!res.ok) {
        throw await res.json()
      }

      // redirect to the new org subdomain
      let created = await res.json()

      window.location = `${window.location.protocol}//${created?.organization?.domain}`
    } catch (e) {
      setSubmitted(false)
      if (e.fieldErrors) {
        const errors = {}
        for (const error of e.fieldErrors) {
          errors[error.fieldName.toLowerCase()] =
            error.errors[0] || 'invalid value'
        }
        setErrors(errors)
      } else {
        setError(e.message)
      }
    }

    return false
  }

  const notURLSafePattern = /[^\da-zA-Z-]/g

  function getURLSafeDomain(domain) {
    // remove spaces
    domain = domain.split(' ').join('-')
    // remove unsafe characters
    domain = domain.replace(notURLSafePattern, '')
    return domain.toLowerCase()
  }

  return (
    <div className='flex w-full flex-col items-center px-10 py-4'>
      <h1 className='mt-4 text-2xl font-bold leading-snug'>Sign up</h1>
      <h2 className='my-2 text-center text-sm text-gray-500'>
        Get started by creating your account
      </h2>
      <form onSubmit={onSubmit} className='mt-8 flex w-full flex-col'>
        <div className='space-y-3'>
          <div className='w-full'>
            <label
              htmlFor='name'
              className='block text-xs font-medium text-gray-700'
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
              className={`mt-1 block w-full rounded-md  shadow-sm focus:border-blue-500 focus:ring-blue-500 sm:text-sm ${
                errors.name ? 'border-red-500' : 'border-gray-300'
              }`}
            />
            {errors.name && (
              <p className='text-xs text-red-500'>{errors.name}</p>
            )}
          </div>
          <div className='w-full'>
            <label
              htmlFor='password'
              className='text-xs font-medium text-gray-700'
            >
              Password
            </label>
            <input
              required
              id='password'
              type='password'
              onChange={e => {
                setPassword(e.target.value)
                setErrors({})
                setError('')
              }}
              className={`mt-1 block w-full rounded-md shadow-sm focus:border-blue-500 focus:ring-blue-500 sm:text-sm ${
                errors.password ? 'border-red-500' : 'border-gray-300'
              }`}
            />
            {errors.password && (
              <p className='text-xs text-red-500'>{errors.password}</p>
            )}
          </div>
          <div className='w-full'>
            <label
              htmlFor='orgName'
              className='text-2xs font-medium text-gray-700'
            >
              Organization
            </label>
            <input
              required
              id='orgName'
              type='text'
              onChange={e => {
                setOrgName(e.target.value)
                setErrors({})
                setError('')
                if (automaticOrgDomain) {
                  setSubDomain(getURLSafeDomain(e.target.value))
                }
              }}
              className={`mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-blue-500 focus:ring-blue-500 sm:text-sm ${
                errors.org?.name ? 'border-red-500' : 'border-gray-800'
              }`}
            />
            {errors.org?.name && (
              <p className='text-xs text-red-500'>{errors.org?.name}</p>
            )}
          </div>
          <div className='w-full'>
            <label
              htmlFor='orgDoman'
              className='text-2xs font-medium text-gray-700'
            >
              Domain
            </label>
            <div className='shadow-sm" mt-1 flex rounded-md'>
              <input
                required
                name='orgDomain'
                type='text'
                autoComplete='off'
                value={subDomain}
                autoCorrect='off'
                onChange={e => {
                  setSubDomain(getURLSafeDomain(e.target.value))
                  setAutomaticOrgDomain(false) // do not set this automatically once it has been specified
                  setErrors({})
                  setError('')
                }}
                className={`block w-full min-w-0 rounded-l-lg px-3 py-2 shadow-sm focus:border-blue-500 focus:ring-blue-500 sm:text-sm ${
                  errors.domain ? 'border-red-500' : 'border-gray-300'
                }`}
              />
              <span className='inline-flex select-none items-center rounded-r-lg border border-l-0 border-gray-300 bg-gray-50 px-3 text-gray-500 shadow-sm sm:text-xs'>
                .{baseDomain}
              </span>
            </div>
            {errors.domain && (
              <p className='text-xs text-red-500'>{errors.domain}</p>
            )}
          </div>
        </div>
        <button
          type='submit'
          disabled={submitted}
          className='mt-6 mb-4 flex w-full cursor-pointer justify-center rounded-lg border border-transparent bg-blue-500 py-2.5 px-4 font-medium text-white shadow-sm hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2 disabled:pointer-events-none disabled:bg-blue-700/50 sm:text-xs'
        >
          Sign Up
        </button>
        {error && <p className='text-xs text-red-500'>{error}</p>}
        <div className='my-3 text-center text-2xs text-gray-400'>
          By continuing, you agree to Infra&apos;s{' '}
          <a
            className='underline'
            href='https://infrahq.com/terms'
            target='_blank'
            rel='noreferrer'
          >
            Terms of Service
          </a>{' '}
          and{' '}
          <a
            className='underline'
            href='https://infrahq.com/privacy'
            target='_blank'
            rel='noreferrer'
          >
            Privacy Policy
          </a>
          .
        </div>
      </form>
    </div>
  )
}

Signup.layout = page => <Login>{page}</Login>
