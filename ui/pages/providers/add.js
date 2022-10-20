import { Fragment, useEffect, useState } from 'react'
import { Transition, Dialog } from '@headlessui/react'
import { useRouter } from 'next/router'
import copy from 'copy-to-clipboard'
import Head from 'next/head'
import Link from 'next/link'
import { useSWRConfig } from 'swr'
import { InformationCircleIcon, XIcon } from '@heroicons/react/outline'
import Tippy from '@tippyjs/react'
import {
  DuplicateIcon,
  CheckIcon,
  InformationCircleIcon,
  XIcon,
} from '@heroicons/react/outline'

import { providers } from '../../lib/providers'

import Dashboard from '../../components/layouts/dashboard'

function SCIMKeyDialog(props) {
  const [keyCopied, setKeyCopied] = useState(false)

  return (
    <div className='w-full 2xl:m-auto'>
      <h1 className='py-1 font-display text-lg font-medium'>SCIM Access Key</h1>
      <div className='space-y-4'>
        <section>
          {props.errorMsg === '' ? (
            <>
              <div className='mb-2'>
                <p className='mt-1 text-sm text-gray-500'>
                  Use this access key to configure your identity provider for
                  inbound SCIM provisioning
                </p>
              </div>
              <div className='group relative my-4 flex'>
                <pre className='w-full overflow-auto rounded-lg bg-gray-50 px-5 py-4 text-xs leading-normal text-gray-800'>
                  {props.accessKey}
                </pre>
                <button
                  className={`absolute right-2 top-2 rounded-md border border-black/10 bg-white px-2 py-2 text-black/40 backdrop-blur-xl hover:text-black/70`}
                  onClick={() => {
                    copy(key)
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
            </>
          ) : (
            <div
              class='mb-4 rounded-lg bg-red-100 p-4 text-sm text-red-700 dark:bg-red-200 dark:text-red-800'
              role='alert'
            >
              <span class='font-medium'>Error:</span> {props.errorMsg}
            </div>
          )}
        </section>

        {/* Finish */}
        <section className={`my-10 flex justify-between`}>
          <Link href='/providers'>
            <a className='flex-none items-center self-center rounded-md border border-transparent bg-black px-4 py-2 text-2xs font-medium text-white shadow-sm hover:bg-gray-800'>
              Finish
            </a>
          </Link>
        </section>
      </div>
    </div>
  )
}

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
  const [enableSCIM, setEnableSCIM] = useState(false)
  const [privateKey, setPrivateKey] = useState('')
  const [clientEmail, setClientEmail] = useState('')
  const [domainAdminEmail, setDomainAdminEmail] = useState('')
  const [error, setError] = useState('')
  const [errors, setErrors] = useState({})
  const [name, setName] = useState('')
  const [scimAccessKey, setSCIMAccessKey] = useState('')
  const [keyDialogOpen, setKeyDialogOpen] = useState(false)

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
              name: name.trim(),
              url,
              clientID,
              clientSecret,
              kind,
              api,
            }),
          })

          const data = await jsonBody(res)

          createSCIMAccessKey(data.id, data.name)

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

    if (!enableSCIM) {
      router.replace('/providers')
    }

    return false
  }

  async function createSCIMAccessKey(id, name) {
    try {
      const res = await fetch('/api/access-keys', {
        method: 'POST',
        body: JSON.stringify({
          userID: id,
          name: name + '-scim',
          ttl: '87600h',
          extensionDeadline: '720h',
        }),
      })

      const data = await res.json()

      if (!res.ok) {
        throw data
      }

      setSCIMAccessKey(data.accessKey)
    } catch (e) {
      setError(e.message)
    }
    setKeyDialogOpen(true)
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
      <header className='my-6 flex items-center justify-between'>
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
                    <SCIMKeyDialog accessKey={scimAccessKey} errorMsg={error} />
                  </Dialog.Panel>
                </Transition.Child>
              </div>
            </div>
          </Dialog>
        </Transition.Root>
      </header>
      <div className='flex items-center justify-between'>
        <h1 className='my-6 py-1 font-display text-xl font-medium'>
          Connect Provider
        </h1>
        <Link href='/providers'>
          <a>
            <XIcon
              className='h-5 w-5 text-gray-500 hover:text-gray-800'
              aria-hidden='true'
            />
          </a>
        </Link>
      </div>
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
            <div className='mt-3 space-y-3'>
              <div>
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
                {errors.name && (
                  <p className='my-1 text-xs text-red-500'>{errors.name}</p>
                )}
              </div>

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
                  {errors.url && (
                    <p className='my-1 text-xs text-red-500'>{errors.url}</p>
                  )}
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
                  onKeyDown={e => {
                    if (e.key === 'Escape' || e.key === 'Esc') {
                      e.preventDefault()
                    }
                  }}
                  className={`mt-1 block w-full rounded-md shadow-sm focus:border-blue-500 focus:ring-blue-500 sm:text-sm ${
                    errors.clientid ? 'border-red-500' : 'border-gray-300'
                  }`}
                />
                {errors.clientid && (
                  <p className='my-1 text-xs text-red-500'>{errors.clientid}</p>
                )}
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
                  <p className='my-1 text-xs text-red-500'>
                    {errors.clientsecret}
                  </p>
                )}
              </div>
              <div>
                <input
                  id='scim-checkbox'
                  type='checkbox'
                  value=''
                  class='h-4 w-4 rounded border-gray-300 bg-gray-100 text-blue-600 focus:ring-2 focus:ring-blue-500 dark:border-gray-600 dark:bg-gray-700 dark:ring-offset-gray-800 dark:focus:ring-blue-600'
                  onChange={() => {
                    setEnableSCIM(!enableSCIM)
                  }}
                />
                <label for='scim-checkbox' class='ml-2 text-sm font-medium'>
                  Enable SCIM
                </label>
              </div>
            </div>
          </div>

          {kind === 'google' && (
            <div className='w-full'>
              <div className='mb-1 flex items-center space-x-2 text-xs'>
                <h3 className='text-base font-medium leading-6 text-gray-900'>
                  Optional information for Google Groups
                </h3>
                <a
                  className='text-gray-500 underline hover:text-gray-600'
                  target='_blank'
                  href='https://infrahq.com/docs/identity-providers/google#groups'
                  rel='noreferrer'
                >
                  learn more
                </a>
              </div>
              <div className='mt-3 space-y-3'>
                <div>
                  <label className='flex items-center text-2xs font-medium text-gray-700'>
                    Private Key
                    <Tippy
                      content='upload the private key json file that was created for
                      your service account'
                      className='whitespace-no-wrap z-8 relative w-60 rounded-md bg-black p-2 text-xs text-white shadow-lg'
                      interactive={true}
                      interactiveBorder={20}
                      delay={100}
                      offset={[0, 5]}
                      placement='top-start'
                    >
                      <InformationCircleIcon className='mx-1 h-4 w-4' />
                    </Tippy>
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
                    <p className='my-1 text-xs text-red-500'>
                      {errors.privatekey}
                    </p>
                  )}
                </div>
                <div>
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
                    <p className='my-1 text-xs text-red-500'>
                      {errors.domainadminemail}
                    </p>
                  )}
                </div>
              </div>
            </div>
          )}
          {error && <p className='my-1 text-xs text-red-500'>{error}</p>}

          <div className='flex items-center justify-end space-x-3 pt-5 pb-3'>
            <button
              type='submit'
              className='inline-flex items-center rounded-md border border-transparent bg-black px-4 py-2 text-xs font-medium text-white shadow-sm hover:cursor-pointer hover:bg-gray-800'
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
