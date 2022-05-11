import { useState } from 'react'
import { useRouter } from 'next/router'
import Head from 'next/head'
import Link from 'next/link'
import { useSWRConfig } from 'swr'

import { providers } from '../../../lib/providers'

import FullscreenModal from '../../../components/modals/fullscreen'
import ErrorMessage from '../../../components/error-message'

export default function () {
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
    <FullscreenModal backHref='/providers/add' closeHref='/providers'>
      <Head>
        <title>Add Identity Provider - {kind}</title>
      </Head>
      <div className='w-full max-w-sm'>
        <div className='flex flex-col pt-8 pb-6 px-4 border rounded-lg border-gray-950'>
          <div className='flex flex-row space-x-2'>
            <img src='/providers.svg' className='w-6 h-6' />
            <h1 className='text-base tracking-tight capitalize'>Connect {kind}</h1>
          </div>
          <form onSubmit={onSubmit} className='flex flex-col mt-12'>
            <div className='mb-8'>
              <label className='text-xs uppercase'>NAME YOUR PROVIDER</label>
              <input
                required
                autoFocus
                placeholder='Name'
                onChange={e => setName(e.target.value)}
                className={`w-full bg-transparent border-b border-gray-950 text-name px-px py-3 focus:outline-none focus:border-b focus:border-gray-200 placeholder:italic ${errors.name ? 'border-pink-300' : ''}`}
              />
              {errors.name && <ErrorMessage message={errors.name} />}
            </div>

            <label className='text-xs'>
              Additional details (<a className='text-pink-300 underline' target='_blank' href='https://infrahq.com/docs/guides/identity-providers/okta' rel='noreferrer'>learn more</a>)
            </label>
            <div className='mt-4'>
              <label className='text-xs uppercase'>URL (Domain)</label>
              <input
                required
                placeholder='URL (Domain)'
                value={url}
                onChange={e => setURL(e.target.value)}
                className={`w-full bg-transparent border-b border-gray-950 text-name px-px py-3 focus:outline-none focus:border-b focus:border-gray-200 placeholder:italic ${errors.url ? 'border-pink-300' : ''}`}
              />
              {errors.url && <ErrorMessage message={errors.url} />}
            </div>
            <div className='mt-4'>
              <label className='text-xs uppercase'>Client ID</label>
              <input
                required
                placeholder='Client ID'
                value={clientID}
                onChange={e => setClientID(e.target.value)}
                className={`w-full bg-transparent border-b border-gray-950 text-name px-px py-3 focus:outline-none focus:border-b focus:border-gray-200 placeholder:italic ${errors.clientid ? 'border-pink-300' : ''}`}
              />
              {errors.clientid && <ErrorMessage message={errors.clientid} />}
            </div>
            <div className='mt-4'>
              <label className='text-xs uppercase'>Client Secret</label>
              <input
                required
                type='password'
                placeholder='Client Secret'
                value={clientSecret}
                onChange={e => setClientSecret(e.target.value)}
                className={`w-full bg-transparent border-b border-gray-950 text-name px-px py-3 focus:outline-none focus:border-b focus:border-gray-200 placeholder:italic ${errors.clientsecret ? 'border-pink-300' : ''}`}
              />
              {errors.clientsecret && <ErrorMessage message={errors.clientsecret} />}
            </div>
            <div className='flex flex-row justify-between mt-6 items-center'>
              <Link href='/providers'>
                <a className='uppercase border-0 hover:text-white text-gray-300'>Cancel</a>
              </Link>
              <button
                type='submit'
                disabled={!name || !url || !clientID || !clientSecret}
                className='bg-gradient-to-tr from-indigo-300 to-pink-100 hover:from-indigo-200 hover:to-pink-50 rounded-md p-0.5 text-center disabled:opacity-30'
              >
                <div className='bg-black rounded-md tracking-tight text-sm px-6 py-3 '>
                  Connect Provider
                </div>
              </button>
            </div>
            {error && <ErrorMessage message={error} center />}
          </form>
        </div>
      </div>
    </FullscreenModal>
  )
}
