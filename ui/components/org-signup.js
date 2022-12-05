import { useState } from 'react'

// The OrgSignup component is shared between the base sign-up page and the Google sign-up callback
export default function OrgSignup({
  baseDomain,
  errors,
  setErrors,
  subDomain,
  setSubDomain,
  setOrgName,
  setError,
}) {
  const [automaticOrgDomain, setAutomaticOrgDomain] = useState(true) // track if the user has manually specified the org domain
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
      <div className='w-full'>
        <label htmlFor='orgName' className='text-2xs font-medium text-gray-700'>
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
          className={`mt-1 block w-full rounded-md shadow-sm focus:border-blue-500 focus:ring-blue-500 sm:text-sm ${
            errors['org.name'] ? 'border-red-500' : 'border-gray-300'
          }`}
        />
        {errors['org.name'] && (
          <p className='my-1 text-xs text-red-500'>{errors['org.name']}</p>
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
              errors['org.subdomain'] ? 'border-red-500' : 'border-gray-300'
            }`}
          />
          <span className='inline-flex select-none items-center rounded-r-lg border border-l-0 border-gray-300 bg-gray-50 px-3 text-2xs text-gray-500 shadow-sm'>
            .{baseDomain}
          </span>
        </div>
        {errors['org.subdomain'] && (
          <p className='my-1 text-xs text-red-500'>{errors['org.subdomain']}</p>
        )}
      </div>
    </>
  )
}
