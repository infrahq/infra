import { useState } from 'react'
import { useRouter } from 'next/router'

import { useServerConfig } from '../../lib/serverconfig'
import { saveToVisitedOrgs } from '../../lib/login'
import LoginLayout from '../../components/layouts/login'
import OrgSignup from '../../components/org-signup'

export default function Callback() {
  const router = useRouter()
  const { isReady } = router
  const { code, state } = router.query
  const [orgName, setOrgName] = useState('')
  const [subDomain, setSubDomain] = useState('')
  const [submitted, setSubmitted] = useState(false)
  const [error, setError] = useState('')
  const [errors, setErrors] = useState({})

  const { baseDomain } = useServerConfig()

  async function onSubmit(e) {
    e.preventDefault()
    setError('')
    setSubmitted(true)

    if (state !== window.localStorage.getItem('state')) {
      setError('social login is in an unexpected state, aborted')
    }

    if (!code) {
      setError('missing google authentication code')
    }

    const redirectURL = window.localStorage.getItem('redirectURL')
    if (!redirectURL) {
      setError('could not read redirect, check that you allow cookies')
    }

    if (error !== '') {
      setSubmitted(false)
      return
    }

    try {
      let res = await fetch('/api/signup', {
        method: 'POST',
        body: JSON.stringify({
          social: {
            code,
            redirectURL,
          },
          orgName,
          subDomain,
        }),
      })

      // redirect to the new org subdomain
      let created = await jsonBody(res)
      saveToVisitedOrgs(`${created?.organization?.domain}`, orgName)

      window.localStorage.removeItem('redirectURL')
      window.location = `${window.location.protocol}//${created?.organization?.domain}`
    } catch (e) {
      setError(e.message)
    }
    setSubmitted(false)
  }

  if (!isReady) {
    return null
  }

  return (
    <div className='flex w-full flex-col items-center px-10 py-4'>
      <h1 className='mt-4 text-2xl font-bold leading-snug'>Sign up</h1>
      <h2 className='my-2 text-center text-sm text-gray-500'>
        Name your Organization
      </h2>
      <form onSubmit={onSubmit} className='mt-8 flex w-full flex-col'>
        <OrgSignup
          baseDomain={baseDomain}
          subDomain={subDomain}
          setSubDomain={setSubDomain}
          setOrgName={setOrgName}
          errors={errors}
          setErrors={setErrors}
          setError={setError}
        />
        <button
          type='submit'
          disabled={submitted}
          className='mt-6 mb-4 flex w-full cursor-pointer justify-center rounded-lg border border-transparent bg-blue-500 py-2.5 px-4 text-sm font-medium text-white shadow-sm hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2 disabled:pointer-events-none disabled:bg-blue-700/50'
        >
          Sign Up
        </button>
        {error && (
          <p className='my-1 text-xs text-red-500'>sign-up failed: {error}</p>
        )}
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

Callback.layout = page => <LoginLayout>{page}</LoginLayout>
