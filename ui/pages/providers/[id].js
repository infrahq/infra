import { useEffect, useState } from 'react'
import { useRouter } from 'next/router'
import Head from 'next/head'
import useSWR, { useSWRConfig } from 'swr'

import ErrorMessage from '../../components/error-message'
import Dashboard from '../../components/layouts/dashboard'

export default function ProvidersEditDetails() {
  const router = useRouter()

  const id = router.query.id

  const { mutate } = useSWRConfig()
  const { data: provider } = useSWR(`/api/providers/${id}`)

  const [name, setName] = useState('')
  const [clientSecret, setClientSecret] = useState('***********')
  const [error, setError] = useState('')
  const [errors, setErrors] = useState({})

  useEffect(() => {
    setName(provider?.name)
  }, [provider])

  async function onSubmit(e) {
    e.preventDefault()

    setErrors({})
    setError('')

    try {
      await mutate('/api/providers', async () => {
        const res = await fetch(`/api/providers/${id}`, {
          method: 'PATCH',
          body: JSON.stringify({
            id,
            name,
            clientSecret,
          }),
        })

        const data = await res.json()

        if (!res.ok) {
          throw data
        }

        return {}
      })
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

      return false
    }

    router.replace('/providers')

    return false
  }

  return (
    <div className='mx-auto w-full max-w-2xl'>
      <Head>
        <title>Edit Identity Provider</title>
      </Head>
      <header className='my-6 flex items-center justify-between'>
        <h1 className='py-1 font-display text-xl font-medium'>Edit provider</h1>
      </header>
      <div className='flex w-full flex-col'>
        <form onSubmit={onSubmit} className='mb-6 space-y-8'>
          {/* Overview */}
          <div>
            <label className='text-2xs font-medium text-gray-700'>Name</label>
            <input
              type='text'
              value={name}
              onChange={e => {
                setName(e.target.value)
                setErrors({})
                setError('')
              }}
              className={`mt-1 block w-full rounded-md shadow-sm focus:border-blue-500 focus:ring-blue-500 sm:text-sm ${
                errors.name ? 'border-red-500' : 'border-gray-300'
              }`}
            />
            {errors.name && <ErrorMessage message={errors.name} />}
          </div>
          <div className='w-full'>
            <div className='mt-6 space-y-3'>
              {provider?.kind !== 'google' && (
                <div>
                  <label className='text-2xs font-medium text-gray-700'>
                    URL (Domain)
                  </label>
                  <input
                    type='text'
                    value={provider?.url}
                    readOnly
                    className={`mt-1 block w-full rounded-md border-gray-300 bg-gray-200 text-gray-600 shadow-sm focus:border-gray-300 focus:ring-0 sm:text-sm`}
                  />
                </div>
              )}

              <div>
                <label className='text-2xs font-medium text-gray-700'>
                  Client ID
                </label>
                <input
                  readOnly
                  type='search'
                  value={provider?.clientID}
                  className={`mt-1 block w-full rounded-md border-gray-300 bg-gray-200 text-gray-600 shadow-sm focus:border-gray-300 focus:ring-0 sm:text-sm`}
                />
              </div>

              <div>
                <label className='text-2xs font-medium text-gray-700'>
                  Client Secret
                </label>
                <input
                  type='password'
                  value={clientSecret}
                  onFocus={() => setClientSecret('')}
                  onChange={e => {
                    setClientSecret(e.target.value)
                    setErrors({})
                    setError('')
                  }}
                  className={`mt-1 block w-full rounded-md shadow-sm focus:border-blue-500 focus:ring-blue-500 sm:text-sm ${
                    errors.clientsecret ? 'border-red-500' : 'border-gray-300'
                  }`}
                />
                {errors.clientsecret && (
                  <ErrorMessage message={errors.clientsecret} />
                )}
              </div>
            </div>
          </div>

          <div className='flex items-center justify-between'>
            <div>{error && <ErrorMessage message={error} />}</div>
            <div className='pt-5 pb-3'>
              <button
                type='submit'
                className='inline-flex items-center rounded-md border border-transparent bg-black px-4 py-2 text-xs font-medium text-white shadow-sm hover:bg-gray-800'
              >
                Save
              </button>
            </div>
          </div>
        </form>
      </div>
    </div>
  )
}

ProvidersEditDetails.layout = page => <Dashboard> {page}</Dashboard>
