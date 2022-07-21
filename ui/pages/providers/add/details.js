import { useState } from 'react'
import { useRouter } from 'next/router'
import Head from 'next/head'
import Link from 'next/link'
import { useSWRConfig } from 'swr'

import Fullscreen from '../../../components/layouts/fullscreen'
import ErrorMessage from '../../../components/error-message'

export default function ProvidersAddDetails() {
  const router = useRouter()
  const { kind } = router.query

  const { mutate } = useSWRConfig()

  const [url, setURL] = useState(kind === 'google' ? 'accounts.google.com' : '')
  const [clientID, setClientID] = useState('')
  const [clientSecret, setClientSecret] = useState('')
  const [privateKey, setPrivateKey] = useState('')
  const [clientEmail, setClientEmail] = useState('')
  const [domainAdminEmail, setDomainAdminEmail] = useState('')
  const [error, setError] = useState('')
  const [errors, setErrors] = useState({})
  const [name, setName] = useState(kind)

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
    <div className='px-3 pt-8 pb-3'>
      <Head>
        <title>Add Identity Provider - {kind}</title>
      </Head>
      <header className='flex flex-row items-center px-2'>
        <img
          alt='providers icon'
          src='/providers.svg'
          className='mr-2 mt-0.5 h-6 w-6'
        />
        <h1 className='text-2xs capitalize'>Connect {kind}</h1>
      </header>
      <form onSubmit={onSubmit} className='mt-12 flex flex-col'>
        <div className='mb-8'>
          <label className='text-3xs uppercase text-gray-400'>
            Name your provider
          </label>
          <input
            required
            type='search'
            placeholder='choose a name for your identity provider'
            value={name}
            onChange={e => setName(e.target.value)}
            className={`w-full border-b border-gray-800 bg-transparent px-px py-3 text-3xs placeholder:italic focus:border-b focus:border-gray-200 focus:outline-none ${
              errors.name ? 'border-pink-500/60' : ''
            }`}
          />
          {errors.name && <ErrorMessage message={errors.name} />}
        </div>
        <label className='text-2xs text-white/90'>
          Additional details{' '}
          <a
            className='text-violet-100 underline'
            target='_blank'
            href={docLink()}
            rel='noreferrer'
          >
            learn more
          </a>
        </label>
        {kind !== 'google' && (
          <div className='mt-4'>
            <label className='text-3xs uppercase text-gray-400'>
              URL (Domain)
            </label>
            <input
              required
              autoFocus
              placeholder='domain or URL'
              value={url}
              onChange={e => setURL(e.target.value)}
              className={`w-full border-b border-gray-800 bg-transparent px-px py-3 text-3xs placeholder:italic focus:border-b focus:border-gray-200 focus:outline-none ${
                errors.url ? 'border-pink-500/60' : ''
              }`}
            />
            {errors.url && <ErrorMessage message={errors.url} />}
          </div>
        )}
        <div className='mt-4'>
          <label className='text-3xs uppercase text-gray-400'>Client ID</label>
          <input
            required
            placeholder='client ID'
            type='search'
            value={clientID}
            onChange={e => setClientID(e.target.value)}
            className={`w-full border-b border-gray-800 bg-transparent px-px py-3 text-3xs placeholder:italic focus:border-b focus:border-gray-200 focus:outline-none ${
              errors.clientid ? 'border-pink-500/60' : ''
            }`}
          />
          {errors.clientid && <ErrorMessage message={errors.clientid} />}
        </div>
        <div className='my-4'>
          <label className='text-3xs uppercase text-gray-400'>
            Client Secret
          </label>
          <input
            required
            type='password'
            placeholder='client secret'
            value={clientSecret}
            onChange={e => setClientSecret(e.target.value)}
            className={`w-full border-b border-gray-800 bg-transparent px-px py-3 text-3xs placeholder:italic focus:border-b focus:border-gray-200 focus:outline-none ${
              errors.clientsecret ? 'border-pink-500/60' : ''
            }`}
          />
          {errors.clientsecret && (
            <ErrorMessage message={errors.clientsecret} />
          )}
        </div>
        {kind === 'google' && (
          <div>
            <label className='text-2xs text-white/90'>
              Optional details{' '}
              <a
                className='text-violet-100 underline'
                target='_blank'
                href='https://infrahq.com/docs/identity-providers/google#groups' /* TODO: make sure this link works*/
                rel='noreferrer'
              >
                learn more
              </a>
            </label>
            <div className='mt-4'>
              <label className='text-3xs uppercase text-gray-400'>
                Private Key
              </label>
              <input
                type='file'
                onChange={parseGoogleCredentialFile}
                className={`w-full border-b border-gray-800 bg-transparent px-px py-3 text-3xs placeholder:italic focus:border-b focus:border-gray-200 focus:outline-none ${
                  errors.privatekey ? 'border-pink-500/60' : ''
                }`}
              />
              {errors.privatekey && (
                <ErrorMessage message={errors.privatekey} />
              )}
            </div>
            <div className='mt-4'>
              <label className='text-3xs uppercase text-gray-400'>
                Workspace Domain Admin
              </label>
              <input
                placeholder='domain admin email'
                spellCheck='false'
                type='email'
                value={domainAdminEmail}
                onChange={e => setDomainAdminEmail(e.target.value)}
                className={`w-full border-b border-gray-800 bg-transparent px-px py-3 text-3xs placeholder:italic focus:border-b focus:border-gray-200 focus:outline-none ${
                  errors.domainadminemail ? 'border-pink-500/60' : ''
                }`}
              />
              {errors.domainadminemail && (
                <ErrorMessage message={errors.domainadminemail} />
              )}
            </div>
          </div>
        )}
        <div className='mt-6 flex flex-row items-center justify-end'>
          <Link href='/providers'>
            <a className='border-0 px-6 py-3 text-2xs uppercase text-gray-400 hover:text-white focus:text-white focus:outline-none'>
              Cancel
            </a>
          </Link>
          <button
            type='submit'
            disabled={!name || !url || !clientID || !clientSecret}
            className='rounded-md border border-violet-300 px-5 py-2.5 text-center text-2xs text-violet-100 disabled:opacity-30'
          >
            Connect Provider
          </button>
        </div>
        {error && <ErrorMessage message={error} center />}
      </form>
    </div>
  )
}

ProvidersAddDetails.layout = page => (
  <Fullscreen backHref='/providers/add' closeHref='/providers'>
    {page}
  </Fullscreen>
)
