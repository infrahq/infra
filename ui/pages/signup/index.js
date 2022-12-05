import { useState } from 'react'

import { useServerConfig } from '../../lib/serverconfig'
import { saveToVisitedOrgs } from '../../lib/login'

import Login from '../../components/layouts/login'
import Providers, { oidcSignup } from '../../components/providers'
import OrgSignup from '../../components/org-signup'

export default function Signup() {
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const [orgName, setOrgName] = useState('')
  const [subDomain, setSubDomain] = useState('')
  const [submitted, setSubmitted] = useState(false)
  const [error, setError] = useState('')
  const [errors, setErrors] = useState({})

  const { baseDomain, google } = useServerConfig()

  async function onSubmit(e) {
    e.preventDefault()

    setSubmitted(true)

    try {
      let res = await fetch('/api/signup', {
        method: 'POST',
        body: JSON.stringify({
          org: {
            username,
            password,
            orgName,
            subDomain,
          },
        }),
      })

      // redirect to the new org subdomain
      let created = await jsonBody(res)

      window.location = `${window.location.protocol}//${created?.organization?.domain}`
      saveToVisitedOrgs(`${created?.organization?.domain}`, orgName)
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

  return (
    <div className='flex w-full flex-col items-center px-10 py-4'>
      <h1 className='mt-4 text-2xl font-bold leading-snug'>Sign up</h1>
      {google !== undefined && (
        <>
          <Providers
            providers={[google]}
            authnFunc={oidcSignup}
            buttonPrompt={'Sign up with'}
          />
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
                setUsername(e.target.value)
                setErrors({})
                setError('')
              }}
              className={`mt-1 block w-full rounded-md  shadow-sm focus:border-blue-500 focus:ring-blue-500 sm:text-sm ${
                errors.name ? 'border-red-500' : 'border-gray-300'
              }`}
            />
            {errors.name && (
              <p className='my-1 text-xs text-red-500'>{errors.name}</p>
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
              <p className='my-1 text-xs text-red-500'>{errors.password}</p>
            )}
          </div>
          <OrgSignup
            baseDomain={baseDomain}
            subDomain={subDomain}
            setSubDomain={setSubDomain}
            setOrgName={setOrgName}
            errors={errors}
            setErrors={setErrors}
            setError={setError}
          />
        </div>
        <button
          type='submit'
          disabled={submitted}
          className='mt-6 mb-4 flex w-full cursor-pointer justify-center rounded-lg border border-transparent bg-blue-500 py-2.5 px-4 text-sm font-medium text-white shadow-sm hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2 disabled:pointer-events-none disabled:bg-blue-700/50'
        >
          Sign Up
        </button>
        {error && <p className='my-1 text-xs text-red-500'>{error}</p>}
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
