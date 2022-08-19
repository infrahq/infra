import { useState } from 'react'

import Login from '../../components/layouts/login'
import ErrorMessage from '../../components/error-message'

export default function Signup() {
  const [name, setName] = useState('')
  const [password, setPassword] = useState('')
  const [confirmPassword, setConfirmPassword] = useState('')
  const [orgName, setOrgName] = useState('')
  const [subDomain, setSubDomain] = useState('')
  const [automaticOrgDomain, setAutomaticOrgDomain] = useState(true) // track if the user has manually specified the org domain
  const [submitted, setSubmitted] = useState(false)
  const [error, setError] = useState('')
  const [errors, setErrors] = useState({})

  async function onSubmit(e) {
    e.preventDefault()

    setSubmitted(true)

    if (password !== confirmPassword) {
      setErrors({
        confirmPassword: 'passwords do not match',
      })
      setSubmitted(false)
      return false
    }

    try {
      // signup
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

    setSubmitted(false)

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
    <>
      <h1 className='text-base font-bold leading-snug'>Welcome to Infra</h1>
      <h2 className='my-1.5 max-w-md text-center text-xs text-gray-400'>
        Set up your admin user to get started.
      </h2>
      <form onSubmit={onSubmit} className='flex w-full max-w-sm flex-col'>
        <div className='my-2 w-full'>
          <label htmlFor='name' className='text-3xs uppercase text-gray-500'>
            Email
          </label>
          <input
            autoFocus
            id='name'
            placeholder='enter your email'
            onChange={e => {
              setName(e.target.value)
              setErrors({})
              setError('')
            }}
            className={`mb-1 w-full border-b border-gray-800 bg-transparent px-px py-2 text-2xs placeholder:italic focus:border-b focus:outline-none focus:ring-gray-200 ${
              errors.name ? 'border-pink-500/60' : ''
            }`}
          />
          {errors.name && <ErrorMessage message={errors.name} />}
        </div>
        <div className='my-2 w-full'>
          <label
            htmlFor='password'
            className='text-3xs uppercase text-gray-500'
          >
            Password
          </label>
          <input
            id='password'
            type='password'
            placeholder='enter your password'
            onChange={e => {
              setPassword(e.target.value)
              setErrors({})
              setError('')
            }}
            className={`mb-1 w-full border-b border-gray-800 bg-transparent px-px py-2 text-2xs placeholder:italic focus:border-b focus:outline-none focus:ring-gray-200 ${
              errors.password ? 'border-pink-500/60' : ''
            }`}
          />
          {errors.password && <ErrorMessage message={errors.password} />}
        </div>
        <div className='my-2 w-full'>
          <label
            htmlFor='password'
            className='text-3xs uppercase text-gray-500'
          >
            Confirm Password
          </label>
          <input
            required
            id='confirmPassword'
            type='password'
            placeholder='confirm your password'
            onChange={e => {
              setConfirmPassword(e.target.value)
              setErrors({})
              setError('')
            }}
            className={`mb-1 w-full border-b border-gray-800 bg-transparent px-px py-2 text-2xs placeholder:italic focus:border-b focus:outline-none focus:ring-gray-200 ${
              errors.confirmPassword ? 'border-pink-500/60' : ''
            }`}
          />
          {errors.confirmPassword && (
            <ErrorMessage message={errors.confirmPassword} />
          )}
        </div>
        <div className='my-2 w-full pt-6 text-2xs uppercase leading-none text-gray-400'>
          Organization
        </div>
        <div className='my-2 w-full'>
          <label htmlFor='orgName' className='text-3xs uppercase text-gray-500'>
            Name
          </label>
          <input
            required
            id='orgName'
            placeholder='name your organization'
            onChange={e => {
              setOrgName(e.target.value)
              setErrors({})
              setError('')
              if (automaticOrgDomain) {
                setSubDomain(getURLSafeDomain(e.target.value))
              }
            }}
            className={`mb-1 w-full border-b border-gray-800 bg-transparent px-px py-2 text-2xs placeholder:italic focus:border-b focus:outline-none focus:ring-gray-200 ${
              errors.org?.name ? 'border-pink-500/60' : ''
            }`}
          />
        </div>
        <div className='my-2 w-full'>
          <label
            htmlFor='orgDoman'
            className='text-3xs uppercase text-gray-500'
          >
            Domain
          </label>
          <div className='flex flex-wrap'>
            <input
              required
              name='orgDomain'
              placeholder='your-domain'
              value={subDomain}
              onChange={e => {
                setSubDomain(getURLSafeDomain(e.target.value))
                setAutomaticOrgDomain(false) // do not set this automatically once it has been specified
                setErrors({})
                setError('')
              }}
              className={`mb-1 w-2/3 border-b border-gray-800 bg-transparent px-px py-2 text-2xs placeholder:italic focus:border-b focus:outline-none focus:ring-gray-200 ${
                errors.org?.subDomain ? 'border-pink-500/60' : ''
              }`}
            />
            <div className='mb-1 w-1/3 border border-gray-800 py-2 text-center text-2xs text-gray-500'>
              .{window.location.host}
            </div>
            {errors.domain && <ErrorMessage message={errors.domain} />}
          </div>
        </div>
        <button
          disabled={
            !name ||
            !password ||
            !confirmPassword ||
            !orgName ||
            !subDomain ||
            submitted
          }
          className='my-2 rounded-lg border border-violet-300 px-4 py-3 text-2xs text-violet-100 hover:border-violet-100 disabled:pointer-events-none disabled:opacity-30'
        >
          Get Started
        </button>
        {error && <ErrorMessage message={error} center />}
      </form>
    </>
  )
}

Signup.layout = page => <Login>{page}</Login>
