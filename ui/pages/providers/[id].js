import { Fragment, useEffect, useRef, useState } from 'react'
import { Transition, Dialog } from '@headlessui/react'
import { useRouter } from 'next/router'
import copy from 'copy-to-clipboard'
import Head from 'next/head'
import Link from 'next/link'
import useSWR, { useSWRConfig } from 'swr'
import { DuplicateIcon, CheckIcon } from '@heroicons/react/outline'
import dayjs from 'dayjs'

import Dashboard from '../../components/layouts/dashboard'
import RemoveButton from '../../components/remove-button'
import Notification from '../../components/notification'

function SCIMKeyDialog(props) {
  const [scimAccessKey, setSCIMAccessKey] = useState('')
  const [error, setError] = useState('')
  const [keyCopied, setKeyCopied] = useState(false)

  async function onSubmit(e) {
    e.preventDefault()

    try {
      let keyName = props.provider.name + '-scim'

      // delete any existing access key for this provider
      await fetch(`/api/access-keys?name=${keyName}`, {
        method: 'DELETE',
      })

      // generate the new key
      const res = await fetch('/api/access-keys', {
        method: 'POST',
        body: JSON.stringify({
          userID: props.provider.id,
          name: keyName,
          ttl: '87600h',
          extensionDeadline: '720h',
        }),
      })

      const data = await jsonBody(res)

      setSCIMAccessKey(data.accessKey)
    } catch (e) {
      setError(e.message)
    }
  }

  return (
    <div className='w-full 2xl:m-auto'>
      <h1 className='py-1 font-display text-lg font-medium'>SCIM Access Key</h1>
      <div className='space-y-4'>
        {scimAccessKey === '' && error === '' ? (
          <section>
            <form onSubmit={onSubmit} className='flex flex-col space-y-4'>
              <div className='mb-4 flex flex-col'>
                <div className='relative mt-4'>
                  <h2 className='mt-5 text-sm'>
                    Generating a new SCIM access key will revoke any existing
                    SCIM access key for this identity provider.
                  </h2>
                  <h2 className='mt-5 text-sm'>Do you wish to continue?</h2>
                </div>
              </div>
              <div className='flex flex-row items-center justify-end space-x-3'>
                <button
                  type='button'
                  onClick={() => props.setOpen(false)}
                  className='inline-flex items-center rounded-md border border-gray-300 bg-white px-4 py-2 text-xs font-medium text-gray-700 shadow-sm hover:bg-gray-100'
                >
                  Cancel
                </button>
                <button
                  type='submit'
                  className='inline-flex items-center rounded-md border border-transparent bg-black px-4 py-2 text-xs font-medium text-white shadow-sm hover:bg-gray-800'
                >
                  Continue
                </button>
              </div>
            </form>
          </section>
        ) : (
          <>
            <section>
              <div className='mb-2'>
                <p className='mt-1 text-sm text-gray-500'>
                  Use this access key to configure your identity provider for
                  inbound SCIM provisioning
                </p>
              </div>
              <div className='group relative my-4 flex'>
                <pre className='w-full overflow-auto rounded-lg bg-gray-50 px-5 py-4 text-xs leading-normal text-gray-800'>
                  {scimAccessKey}
                </pre>
                <button
                  className={`absolute right-2 top-2 rounded-md border border-black/10 bg-white px-2 py-2 text-black/40 backdrop-blur-xl hover:text-black/70`}
                  onClick={() => {
                    copy(scimAccessKey)
                    setKeyCopied(true)
                    setTimeout(() => setKeyCopied(false), 2000)
                  }}
                >
                  {keyCopied ? (
                    <CheckIcon className='h-4 w-4 text-green-500' />
                  ) : (
                    <DuplicateIcon className='h-4 w-4' />
                  )}
                </button>
              </div>
            </section>

            {/* Finish */}
            <section className={`my-10 flex justify-between`}>
              <Link href='/providers'>
                <a className='flex-none items-center self-center rounded-md border border-transparent bg-black px-4 py-2 text-2xs font-medium text-white shadow-sm hover:bg-gray-800'>
                  Finish
                </a>
              </Link>
            </section>
          </>
        )}
      </div>
    </div>
  )
}

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
  const [clientSecret, setClientSecret] = useState('')
  const [errors, setErrors] = useState({})
  const [showNotification, setShowNotification] = useState(false)
  const [keyDialogOpen, setKeyDialogOpen] = useState(false)

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
    setShowNotification(false)
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

        await jsonBody(res)

        setShowNotification(true)
        timerRef.current = setTimeout(() => {
          setShowNotification(false)
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
    setClientSecret('')

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
          <h1 className='flex max-w-[75%] truncate py-1 font-display text-xl font-medium'>
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
            {/* SCIM key dialog */}
            <Transition.Root show={keyDialogOpen} as={Fragment}>
              <Dialog
                as='div'
                className='relative z-50'
                onClose={() => router.replace('/providers')}
              >
                <Transition.Child
                  as={Fragment}
                  enter='ease-out duration-150'
                  enterFrom='opacity-0'
                  enterTo='opacity-100'
                  leave='ease-in duration-100'
                  leaveFrom='opacity-100'
                  leaveTo='opacity-0'
                >
                  <div className='fixed inset-0 bg-white bg-opacity-75 backdrop-blur-xl transition-opacity' />
                </Transition.Child>
                <div className='fixed inset-0 z-10 overflow-y-auto'>
                  <div className='flex min-h-full items-end justify-center p-4 text-center sm:items-center sm:p-0'>
                    <Transition.Child
                      as={Fragment}
                      enter='ease-out duration-150'
                      enterFrom='opacity-0 translate-y-4 sm:translate-y-0 sm:scale-95'
                      enterTo='opacity-100 translate-y-0 sm:scale-100'
                      leave='ease-in duration-100'
                      leaveFrom='opacity-100 translate-y-0 sm:scale-100'
                      leaveTo='opacity-0 translate-y-4 sm:translate-y-0 sm:scale-95'
                    >
                      <Dialog.Panel className='relative w-full transform overflow-hidden rounded-xl border border-gray-100 bg-white p-8 text-left shadow-xl shadow-gray-300/10 transition-all sm:my-8 sm:max-w-lg'>
                        <SCIMKeyDialog
                          provider={provider}
                          setOpen={setKeyDialogOpen}
                        />
                      </Dialog.Panel>
                    </Transition.Child>
                  </div>
                </div>
              </Dialog>
            </Transition.Root>
          </h1>

          <div className='my-3 flex space-x-2 md:my-0'>
            <button
              onClick={() => setKeyDialogOpen(true)}
              className='inline-flex items-center rounded-md border border-transparent bg-black px-4 py-2 text-xs font-medium text-white shadow-sm hover:cursor-pointer hover:bg-gray-800'
            >
              Generate SCIM Access Key
            </button>
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
                onKeyDown={e => {
                  if (e.key === 'Escape' || e.key === 'Esc') {
                    e.preventDefault()
                  }
                }}
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
                placeholder='*********************'
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
                    clientSecret.length === 0 && name === provider?.name
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
        setShow={setShowNotification}
        text={
          <div>
            <span className='break-all font-bold'>{provider?.name}</span> was
            updated successfully
          </div>
        }
        setClearNotification={() => clearTimer()}
      />
    </div>
  )
}

ProvidersEditDetails.layout = page => <Dashboard> {page}</Dashboard>
