import { useState } from 'react'
import { useRouter } from 'next/router'
import Head from 'next/head'
import { SwitchHorizontalIcon } from '@heroicons/react/outline'
import { useSWRConfig } from 'swr'

import { providers } from '../../../lib/providers'

import FullscreenModal from '../../../components/modals/fullscreen'

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

  if (!providers.find(p => p.name.toLowerCase() === kind)) {
    router.replace('/providers/add')
    return null
  }

  async function onSubmit (e) {
    e.preventDefault()

    try {
      setErrors({})
      setError('')
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
      <div className='flex flex-col mb-10 w-full max-w-sm'>
        <h1 className='text-xl font-bold tracking-tight text-center'>Add Identity Provider</h1>
        <h2 className='mt-1 mb-10 text-gray-300 text-center'>Provide your identity provider's details.</h2>
        <div className='flex items-center space-x-4 mx-auto select-none'>
          <img className='h-4' src={`/providers/${kind}.svg`} /><SwitchHorizontalIcon className='w-4 h-4 text-gray-500' /><img src='/icon-light.svg' />
        </div>
        <form onSubmit={onSubmit} className='flex flex-col my-12'>
          <label className='text-xs'>Choose a name</label>
          <input
            required
            autoFocus
            placeholder='Name'
            onChange={e => setName(e.target.value)}
            className={`bg-purple-100/5 border border-zinc-800 text-sm px-5 mt-2 py-2.5 rounded-full focus:outline-none focus:ring focus:ring-cyan-600 ${errors.name ? 'border-pink-500' : ''}`}
          />
          {errors.name && <p className='px-4 mb-1 text-sm text-pink-500'>{errors.name}</p>}

          <label className='text-xs mt-4'>
            Additional details (<a className='text-cyan-400 underline' target='_blank' href='https://infrahq.com/docs/guides/identity-providers/okta' rel='noreferrer'>learn more</a>)
          </label>
          <input
            required
            placeholder='URL (Domain)'
            value={url}
            onChange={e => setURL(e.target.value)}
            className={`bg-purple-100/5 border border-zinc-800 text-sm px-5 mt-2 py-2.5 rounded-full focus:outline-none focus:ring focus:ring-cyan-600 ${errors.url ? 'border-pink-500' : ''}`}
          />
          {errors.url && <p className='px-4 mb-1 text-sm text-pink-500'>{errors.url}</p>}

          <input
            required
            placeholder='Client ID'
            value={clientID}
            onChange={e => setClientID(e.target.value)}
            className={`bg-purple-100/5 border border-zinc-800 text-sm px-5 mt-2 py-2.5 rounded-full focus:outline-none focus:ring focus:ring-cyan-600 ${errors.clientid ? 'border-pink-500' : ''}`}
          />
          {errors.clientid && <p className='px-4 mb-1 text-sm text-pink-500'>{errors.clientid}</p>}

          <input
            required
            type='password'
            placeholder='Client Secret'
            value={clientSecret}
            onChange={e => setClientSecret(e.target.value)}
            className={`bg-purple-100/5 border border-zinc-800 text-sm px-5 mt-2 py-2.5 rounded-full focus:outline-none focus:ring focus:ring-cyan-600 ${errors.clientsecret ? 'border-pink-500' : ''}`}
          />
          {errors.clientsecret && <p className='px-4 mb-1 text-sm text-pink-500'>{errors.clientsecret}</p>}

          <button type='submit' className='bg-gradient-to-tr from-indigo-300 to-pink-100 hover:from-indigo-200 hover:to-pink-50 rounded-full p-0.5 w-full mt-6 text-center'>
            <div className='bg-black rounded-full tracking-tight text-sm px-6 py-3 '>
              Add Identity Provider
            </div>
          </button>
          {error && <p className='mt-2 text-sm text-pink-500 text-center'>{error}</p>}
        </form>
      </div>
    </FullscreenModal>
  )
}
