import { useState } from 'react'
import { useRouter } from 'next/router'
import Head from 'next/head'
import Link from 'next/link'
import { useSWRConfig } from 'swr'

import { providers } from '../../../lib/providers'

import Fullscreen from '../../../components/layouts/fullscreen'
import ErrorMessage from '../../../components/error-message'

export default function ProvidersAddDetails () {
  const router = useRouter()
  const { kind } = router.query

  const { mutate } = useSWRConfig()

  const [url, setURL] = useState('')
  const [clientID, setClientID] = useState('')
  const [clientSecret, setClientSecret] = useState('')
  const [error, setError] = useState('')
  const [errors, setErrors] = useState({})
  const [name, setName] = useState(kind)

  const provider = providers.find(p => p.name.toLowerCase() === kind)

  if (!provider) {
    router.replace('/providers/add')
    return null
  }

  async function onSubmit (e) {
    e.preventDefault()

    setErrors({})
    setError('')

    try {
      await mutate('/v1/providers', async providers => {
        const res = await fetch('/v1/providers', {
          method: 'POST',
          body: JSON.stringify({
            name,
            url,
            clientID,
            clientSecret
          })
        })

        const data = await res.json()

        if (!res.ok) {
          throw data
        }

        return [...(providers || []), data]
      })
    } catch (e) {
      if (e.fieldErrors) {
        const errors = {}
        for (const error of e.fieldErrors) {
          errors[error.fieldName.toLowerCase()] = error.errors[0] || 'invalid value'
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
    <div className='pt-8 px-3 pb-3'>
      <Head>
        <title>Add Identity Provider - {kind}</title>
      </Head>
      <header className='flex flex-row px-2 items-center'>
        <img src='/providers.svg' className='w-6 h-6 mr-2 mt-0.5' />
        <h1 className='text-2xs capitalize'>Connect {kind}</h1>
      </header>
      <form onSubmit={onSubmit} className='flex flex-col mt-12'>
        <div className='mb-8'>
          <label className='text-3xs text-gray-400 uppercase'>Name your provider</label>
          <input
            required
            autoFocus
            placeholder='choose a name for your identity provider'
            onChange={e => setName(e.target.value)}
            className={`w-full bg-transparent border-b border-gray-800 text-3xs px-px py-3 focus:outline-none focus:border-b focus:border-gray-200 placeholder:italic ${errors.name ? 'border-pink-500/60' : ''}`}
          />
          {errors.name && <ErrorMessage message={errors.name} />}
        </div>
        <label className='text-2xs text-white/90'>
          Additional details <a className='text-violet-100 underline' target='_blank' href='https://infrahq.com/docs/guides/identity-providers/okta' rel='noreferrer'>learn more</a>
        </label>
        <div className='mt-4'>
          <label className='text-3xs text-gray-400 uppercase'>URL (Domain)</label>
          <input
            required
            placeholder='domain or URL'
            value={url}
            onChange={e => setURL(e.target.value)}
            className={`w-full bg-transparent border-b border-gray-800 text-3xs px-px py-3 focus:outline-none focus:border-b focus:border-gray-200 placeholder:italic ${errors.url ? 'border-pink-500/60' : ''}`}
          />
          {errors.url && <ErrorMessage message={errors.url} />}
        </div>
        <div className='mt-4'>
          <label className='text-3xs text-gray-400 uppercase'>Client ID</label>
          <input
            required
            placeholder='client ID'
            value={clientID}
            onChange={e => setClientID(e.target.value)}
            className={`w-full bg-transparent border-b border-gray-800 text-3xs px-px py-3 focus:outline-none focus:border-b focus:border-gray-200 placeholder:italic ${errors.clientid ? 'border-pink-500/60' : ''}`}
          />
          {errors.clientid && <ErrorMessage message={errors.clientid} />}
        </div>
        <div className='mt-4'>
          <label className='text-3xs text-gray-400 uppercase'>Client Secret</label>
          <input
            required
            type='password'
            placeholder='client secret'
            value={clientSecret}
            onChange={e => setClientSecret(e.target.value)}
            className={`w-full bg-transparent border-b border-gray-800 text-3xs px-px py-3 focus:outline-none focus:border-b focus:border-gray-200 placeholder:italic ${errors.clientsecret ? 'border-pink-500/60' : ''}`}
          />
          {errors.clientsecret && <ErrorMessage message={errors.clientsecret} />}
        </div>
        <div className='flex flex-row justify-end mt-6 items-center'>
          <Link href='/providers'>
            <a className='uppercase border-0 hover:text-white px-6 py-3 focus:outline-none focus:text-white text-gray-400 text-2xs'>Cancel</a>
          </Link>
          <button
            type='submit'
            disabled={!name || !url || !clientID || !clientSecret}
            className='border border-violet-300 text-2xs text-violet-100 rounded-md px-5 py-2.5 text-center disabled:opacity-30'
          >
            Connect Provider
          </button>
        </div>
        {error && <ErrorMessage message={error} center />}
      </form>
    </div>
  )
}

ProvidersAddDetails.layout = page => <Fullscreen backHref='/providers/add' closeHref='/providers'>{page}</Fullscreen>
