import { useState } from 'react'
import { useRouter } from 'next/router'
import { SwitchHorizontalIcon } from '@heroicons/react/outline'
import Head from 'next/head'

import FullscreenModal from '../../../components/modals/fullscreen'

export default function () {
  const router = useRouter()
  const { kind } = router.query

  const [name, setName] = useState(kind)
  const [url, setURL] = useState('')
  const [clientID, setClientID] = useState('')
  const [clientSecret, setClientSecret] = useState('')

  async function onSubmit (e) {
    e.preventDefault()

    try {
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

      // data
      if (data.code >= 400) {
        if (data.fieldErrors) {

        }

        return false
      }
    } catch (e) {
      console.log(e)
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
        <h2 className='mt-2 mb-10 text-gray-300 text-center'>Provide your identity provider's details.</h2>
        <div className='flex items-center space-x-4 mx-auto select-none'>
          <img className='h-4' src={`/${kind}.svg`} /><SwitchHorizontalIcon className='w-4 h-4 text-gray-500' /><img src='/icon-light.svg' />
        </div>
        <form onSubmit={onSubmit} className='flex flex-col my-12'>
          <input autoFocus placeholder='Name' value={name} onChange={e => setName(e.target.value)} className='bg-purple-100/5 border border-zinc-800 text-sm px-4 my-1 py-2.5 rounded-lg focus:outline-none focus:ring focus:ring-cyan-600' />
          <input autoFocus placeholder='URL (Domain)' value={url} onChange={e => setURL(e.target.value)} className='bg-purple-100/5 border border-zinc-800 text-sm px-4 my-1 py-2.5 rounded-lg focus:outline-none focus:ring focus:ring-cyan-600' />
          <input placeholder='Client ID' value={clientID} onChange={e => setClientID(e.target.value)} className='bg-purple-100/5 border border-zinc-800 text-sm px-4 my-1 py-2.5 rounded-lg focus:outline-none focus:ring focus:ring-cyan-600' />
          <input type='password' value={clientSecret} onChange={e => setClientSecret(e.target.value)} placeholder='Client Secret' className='bg-purple-100/5 border border-zinc-800 text-sm px-4 my-1 py-2 rounded-lg focus:outline-none focus:ring focus:ring-cyan-600' />
          <button type='submit' className='bg-gradient-to-tr from-indigo-300 to-pink-100 hover:from-indigo-200 hover:to-pink-50 rounded-full p-0.5 w-full my-4 text-center'>
            <div className='bg-black rounded-full tracking-tight text-sm px-6 py-3 '>
              Add Identity Provider
            </div>
          </button>
        </form>
      </div>
    </FullscreenModal>
  )
}
