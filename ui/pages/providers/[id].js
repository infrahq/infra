import { useEffect, useRef, useState } from 'react'
import { useRouter } from 'next/router'
import Head from 'next/head'
import Link from 'next/link'
import useSWR, { useSWRConfig } from 'swr'
import dayjs from 'dayjs'

import Dashboard from '../../components/layouts/dashboard'
import RemoveButton from '../../components/remove-button'
import Notification from '../../components/notification'

const CLIENT_SECRET_INIT = '***********'

export default function ProvidersEditDetails() {
  const router = useRouter()
  const id = router.query.id

  const { mutate } = useSWRConfig()
  const { data: provider, mutate: providerMutate } = useSWR(
    `/api/providers/${id}`
  )

  const timerRef = useRef(null)

  const [name, setName] = useState('')
  const [error, setError] = useState('')
  const [clientSecret, setClientSecret] = useState(CLIENT_SECRET_INIT)
  const [errors, setErrors] = useState({})
  const [showNotification, setshowNotification] = useState(false)

  const metadata = [
    { label: 'ID', value: provider?.id, font: 'font-mono' },
    {
      label: 'Created',
      value: provider?.created ? dayjs(provider?.created).fromNow() : '-',
    },
    {
      label: 'Updated',
      value: provider?.updated ? dayjs(provider?.updated).fromNow() : '-',
    },
  ]

  useEffect(() => {
    setName(provider?.name)
  }, [provider])

  useEffect(() => {
    return clearTimer()
  }, [])

  function clearTimer() {
    setshowNotification(false)
    return clearTimeout(timerRef.current)
  }

  async function onSubmit(e) {
    e.preventDefault()

    setErrors({})
    setError('')

    try {
      await mutate('/api/providers', async () => {
        const res = await fetch(`/api/providers/${id}`, {
          method: 'PATCH',
          body: JSON.stringify({
            name,
            clientSecret,
          }),
        })

        const data = await res.json()

        if (!res.ok) {
          throw data
        }

        setshowNotification(true)
        timerRef.current = setTimeout(() => {
          setshowNotification(false)
        }, 5000)

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

    providerMutate()
    setClientSecret(CLIENT_SECRET_INIT)

    return false
  }

  return (
    <div className='mb-10'>
      <Head>
        <title>{provider?.name} - Infra</title>
      </Head>
      {/* Header */}
      <header className='mt-6 mb-12 space-y-4'>
        <div className='flex flex-col justify-between md:flex-row md:items-center'>
          <h1 className='flex truncate py-1 font-display text-xl font-medium'>
            <Link href='/providers'>
              <a className='text-gray-500/75 hover:text-gray-600'>Providers</a>
            </Link>{' '}
            <span className='mx-3 font-light text-gray-400'> / </span>{' '}
            <div className='flex truncate'>
              <div className='mr-2 flex h-8 w-8 flex-none items-center justify-center rounded-md border border-gray-200'>
                <img
                  alt='kubernetes icon'
                  className='h-[18px]'
                  src={`/providers/${provider?.kind}.svg`}
                />
              </div>
              <span className='truncate'>{provider?.name}</span>
            </div>
          </h1>

          <div className='my-3 flex space-x-2 md:my-0'>
            <RemoveButton
              onRemove={async () => {
                await fetch(`/api/providers/${id}`, {
                  method: 'DELETE',
                })

                router.replace('/providers')
              }}
              modalTitle='Remove Identity Provider'
              modalMessage={
                <>
                  Are you sure you want to remove{' '}
                  <span className='font-bold'>{provider?.name}</span>?
                </>
              }
            >
              Remove Provider
            </RemoveButton>
          </div>
        </div>
        {provider && (
          <div className='flex flex-row border-t border-gray-100'>
            {metadata.map(g => (
              <div
                key={g.label}
                className='px-6 py-5 text-left first:pr-6 first:pl-0'
              >
                <div className='text-2xs text-gray-400'>{g.label}</div>
                <span
                  className={`text-sm ${
                    g.font ? g.font : 'font-medium'
                  } text-gray-800`}
                >
                  {g.value}
                </span>
              </div>
            ))}
          </div>
        )}
      </header>
      <div className='my-2.5'>
        {provider && (
          <form onSubmit={onSubmit} className='mb-6 space-y-2'>
            <div>
              <label className='text-2xs font-medium text-gray-700'>Name</label>
              <input
                type='search'
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
              {errors.name && (
                <p className='my-1 text-xs text-red-500'>{errors.name}</p>
              )}{' '}
            </div>

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

            <div>
              <label className='text-2xs font-medium text-gray-700'>
                Client ID
              </label>
              <input
                readOnly
                type='text'
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
                <p className='my-1 text-xs text-red-500'>
                  {errors.clientsecret}
                </p>
              )}
            </div>

            <div className='flex items-center justify-end'>
              {error && <p className='my-1 text-xs text-red-500'>{error}</p>}
              <div className='pt-5 pb-3'>
                <button
                  disabled={
                    clientSecret === CLIENT_SECRET_INIT &&
                    name === provider?.name
                  }
                  type='submit'
                  className='inline-flex items-center rounded-md border border-transparent bg-black px-4 py-2 text-xs font-medium text-white shadow-sm hover:bg-gray-800 disabled:cursor-not-allowed disabled:opacity-30'
                >
                  Save Changes
                </button>
              </div>
            </div>
          </form>
        )}
      </div>
      {/* Notification */}
      <Notification
        show={showNotification}
        setShow={setshowNotification}
        text={`${provider?.name} was successfully updated`}
        setClearNotification={() => clearTimer()}
      />
    </div>
  )
}

ProvidersEditDetails.layout = page => <Dashboard> {page}</Dashboard>
