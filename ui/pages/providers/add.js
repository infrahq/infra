import { useEffect, useState } from 'react'
import { useRouter } from 'next/router'
import Head from 'next/head'
import { useSWRConfig } from 'swr'
import { InformationCircleIcon } from '@heroicons/react/outline'

import { providers } from '../../lib/providers'

import ErrorMessage from '../../components/error-message'
import Dashboard from '../../components/layouts/dashboard'
import Tooltip from '../../components/tooltip'

function Provider({ kind, name, currentKind }) {
  return (
    <div
      className={`${
        kind === currentKind ? 'bg-gray-400/20' : 'bg-white'
      } flex cursor-pointer select-none items-center rounded-lg border border-gray-300 bg-transparent px-3
        py-4 hover:opacity-75`}
    >
      <img
        alt='provider icon'
        className='mr-4 w-6 flex-none'
        src={`/providers/${kind}.svg`}
      />
      <div>
        <h3 className='flex-1 text-2xs'>{name}</h3>
      </div>
    </div>
  )
}

export default function ProvidersAddDetails() {
  const router = useRouter()

  const { type } = router.query

  const { mutate } = useSWRConfig()

  const [kind, setKind] = useState(
    type === undefined ? providers[0].kind : type
  )
  const [url, setURL] = useState('')
  const [clientID, setClientID] = useState('')
  const [clientSecret, setClientSecret] = useState('')
  const [privateKey, setPrivateKey] = useState('')
  const [clientEmail, setClientEmail] = useState('')
  const [domainAdminEmail, setDomainAdminEmail] = useState('')
  const [error, setError] = useState('')
  const [errors, setErrors] = useState({})
  const [name, setName] = useState('')

  useEffect(() => {
    setURL(type === 'google' ? 'accounts.google.com' : '')
  }, [type])

  function docLink() {
    if (kind == 'azure') {
      return 'https://infrahq.com/docs/identity-providers/azure-ad'
    }

    return 'https://infrahq.com/docs/identity-providers/' + kind
  }

  async function onSubmit(e) {
    e.preventDefault()

    setErrors({})
    setError('')

    const api = {
      privateKey,
      clientEmail,
      domainAdminEmail: domainAdminEmail,
    }

    try {
      await mutate(
        '/api/providers',
        async ({ items: providers } = { items: [] }) => {
          const res = await fetch('/api/providers', {
            method: 'POST',
            body: JSON.stringify({
              name,
              url,
              clientID,
              clientSecret,
              kind,
              api,
            }),
          })

          const data = await res.json()

          if (!res.ok) {
            throw data
          }

          return { items: [...providers, data] }
        }
      )
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

  const parseGoogleCredentialFile = e => {
    setErrors({})

    const fileReader = new FileReader()
    fileReader.readAsText(e.target.files[0], 'UTF-8')
    fileReader.onload = e => {
      let errMsg = ''
      try {
        let contents = JSON.parse(e.target.result)

        if (contents.private_key === undefined) {
          errMsg = 'invalid service account key file, no private_key found'
        } else {
          setPrivateKey(contents.private_key)
        }

        if (contents.client_email === undefined) {
          errMsg = 'invalid service account key file, no client_email found'
        } else {
          setClientEmail(contents.client_email)
        }
      } catch (e) {
        errMsg = e.ErrorMessage
        if (e instanceof SyntaxError) {
          errMsg = 'invalid service account key file, must be json'
        }
      }

      if (errMsg !== '') {
        const errors = {}
        errors['privatekey'] = errMsg
        setErrors(errors)
      }
    }
  }

  return (
    <div className='mx-auto w-full max-w-2xl'>
      <Head>
        <title>Add Identity Provider - {kind}</title>
      </Head>
      <h1 className='my-6 py-1 font-display text-xl font-medium'>
        Connect Provider
      </h1>
      <div className='flex w-full flex-col'>
        <form onSubmit={onSubmit} className='mb-6 space-y-8'>
          {/* Overview */}
          <div>
            <h3 className='mb-4 text-base font-medium text-gray-900'>
              Select an identity provider
            </h3>
            <div className='mb-6 grid grid-cols-2 gap-2'>
              {providers.map(
                p =>
                  p.available && (
                    <div
                      key={p.name}
                      onClick={() => {
                        setKind(p.kind)
                        router.replace(`/providers/add?type=${p.kind}`)
                      }}
                    >
                      <Provider {...p} currentKind={kind} />
                    </div>
                  )
              )}
            </div>
          </div>
          <div className='w-full'>
            <div className='mb-1 flex items-center space-x-2 text-xs'>
              <h3 className='text-base font-medium leading-6 text-gray-900'>
                Information
              </h3>
              <a
                className=' text-gray-500 underline hover:text-gray-600'
                target='_blank'
                href={docLink()}
                rel='noreferrer'
              >
                learn more
              </a>
            </div>

            <label className='text-2xs font-medium text-gray-700'>
              Name (optional)
            </label>
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
            <div className='mt-6 space-y-3'>
              {kind !== 'google' && (
                <div>
                  <label className='text-2xs font-medium text-gray-700'>
                    URL (Domain)
                  </label>
                  <input
                    required
                    type='text'
                    value={url}
                    onChange={e => {
                      setURL(e.target.value)
                      setErrors({})
                      setError('')
                    }}
                    className={`mt-1 block w-full rounded-md shadow-sm focus:border-blue-500 focus:ring-blue-500 sm:text-sm ${
                      errors.url ? 'border-red-500' : 'border-gray-300'
                    }`}
                  />
                  {errors.url && <ErrorMessage message={errors.url} />}
                </div>
              )}

              <div>
                <label className='text-2xs font-medium text-gray-700'>
                  Client ID
                </label>
                <input
                  required
                  type='search'
                  value={clientID}
                  onChange={e => {
                    setClientID(e.target.value)
                    setErrors({})
                    setError('')
                  }}
                  className={`mt-1 block w-full rounded-md shadow-sm focus:border-blue-500 focus:ring-blue-500 sm:text-sm ${
                    errors.clientid ? 'border-red-500' : 'border-gray-300'
                  }`}
                />
                {errors.clientid && <ErrorMessage message={errors.clientid} />}
              </div>

              <div>
                <label className='text-2xs font-medium text-gray-700'>
                  Client Secret
                </label>
                <input
                  required
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
                  <ErrorMessage message={errors.clientsecret} />
                )}
              </div>
            </div>
          </div>

          {kind === 'google' && (
            <div className='space-y-1'>
              <div className='py-6'>
                <h3 className='text-base font-medium leading-6 text-gray-900'>
                  Optional information for Google Groups
                </h3>
                <div className='mt-1 flex flex-row items-center space-x-1 text-sm text-gray-500'>
                  <a
                    className='underline hover:text-gray-600'
                    target='_blank'
                    href='https://infrahq.com/docs/identity-providers/google#groups'
                    rel='noreferrer'
                  >
                    learn more
                  </a>
                </div>
              </div>
              <div className='mt-6 grid grid-cols-1 gap-y-6 gap-x-4 sm:grid-cols-6'>
                <div className='sm:col-span-6 lg:col-span-5'>
                  <label className='flex text-2xs font-medium text-gray-700'>
                    Private Key
                    <Tooltip
                      message='upload the private key json file that was created for
                        your service account'
                    >
                      <InformationCircleIcon className='mx-1 h-4 w-4' />
                    </Tooltip>
                  </label>

                  <input
                    type='file'
                    onChange={parseGoogleCredentialFile}
                    className='mt-1 block w-full rounded-md py-[6px] file:mr-4 file:rounded-md file:border-0 file:bg-blue-50
                      file:py-2 file:px-4
                      file:text-sm file:font-semibold
                      file:text-blue-700 hover:file:bg-blue-100
                      sm:text-sm'
                  />
                  {errors.privatekey && (
                    <ErrorMessage message={errors.privatekey} />
                  )}
                </div>
                <div className='sm:col-span-6 lg:col-span-5'>
                  <label className='text-2xs font-medium text-gray-700'>
                    Workspace Domain Admin Email
                  </label>
                  <input
                    spellCheck='false'
                    type='email'
                    value={domainAdminEmail}
                    onChange={e => {
                      setDomainAdminEmail(e.target.value)
                      setErrors({})
                      setError('')
                    }}
                    className={`mt-1 block w-full rounded-md shadow-sm focus:border-blue-500 focus:ring-blue-500 sm:text-sm ${
                      errors.domainadminemail
                        ? 'border-red-500'
                        : 'border-gray-300'
                    }`}
                  />
                  {errors.domainadminemail && (
                    <ErrorMessage message={errors.domainadminemail} />
                  )}
                </div>
              </div>
            </div>
          )}
          {error && <ErrorMessage message={error} center />}

          <div className='flex items-center justify-end space-x-3 pt-5 pb-3'>
            <button
              type='submit'
              className='inline-flex items-center rounded-md border border-transparent bg-black px-4 py-2 text-xs font-medium text-white shadow-sm hover:bg-gray-800'
            >
              Connect Provider
            </button>
          </div>
        </form>
      </div>
    </div>
  )
}

ProvidersAddDetails.layout = page => <Dashboard> {page}</Dashboard>
