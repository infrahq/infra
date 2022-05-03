import { useRouter } from 'next/router'
import { useState } from 'react'
import useSWR, { useSWRConfig } from 'swr'

import { kind } from '../../lib/providers'

import HeaderIcon from '../../components/header-icon'

function oidcLogin ({ id, url, clientID }) {
  window.localStorage.setItem('providerId', id)

  const state = [...Array(10)].map(() => (~~(Math.random() * 36)).toString(36)).join('')
  window.localStorage.setItem('state', state)

  const redirectURL = window.location.origin + '/login/callback'
  window.localStorage.setItem('redirectURL', redirectURL)

  document.location.href = `https://${url}/oauth2/v1/authorize?redirect_uri=${redirectURL}&client_id=${clientID}&response_type=code&scope=openid+email+groups+offline_access&state=${state}`
}

export default function () {
  const { data: providers } = useSWR('/v1/providers', { fallbackData: [] })
  const { mutate } = useSWRConfig()
  const router = useRouter()

  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState('')

  async function onSubmit (e) {
    e.preventDefault()

    try {
      const res = await fetch('/v1/login', {
        method: 'post',
        body: JSON.stringify({
          passwordCredentials: {
            email,
            password
          }
        })
      })

      if (!res.ok) {
        throw await res.json()
      }

      const data = await res.json()

      if (data.passwordUpdateRequired) {
        router.replace({
          pathname: '/login/finish',
          query: { id: res.data.polymorphicId.split(':')[1] }
        })
        return
      }

      mutate('/v1/identities/self', { optimisticData: { name: email } })

      router.replace('/')
    } catch (e) {
      console.error(e)
      setError('Invalid credentials')
    }

    return false
  }

  const kindCount = providers?.items?.map(p => kind(p.url)).reduce((p, c) => {
    p[c] = (p[c] || 0) + 1
    return p
  }, {})

  return (
    <div className='flex flex-col justify-center items-center h-full w-full max-w-md mx-auto mb-48'>
      <HeaderIcon size={12} iconPath='/infra-color.svg' />
      <h1 className='mt-5 text-base font-bold'>Login to Infra</h1>
      <h2 className='text-sm text-center max-w-xs my-2 text-gray-300'>Welcome back. Login with your credentials {providers.length > 0 && 'or via your identity provider.'}</h2>

      {providers?.count > 0 && (
        <>
          <div className='w-full max-w-sm mt-8'>
            {providers?.items?.map(p => (
              <button onClick={() => oidcLogin(p)} key={p.id} className='w-full bg-gradient-to-tr from-indigo-300 to-pink-100 hover:from-indigo-200 hover:to-pink-50 rounded-full p-0.5 my-1.5'>
                <div className='w-full flex flex-col items-center justify-center bg-black rounded-full text-sm px-4 py-4'>
                  {kind(p.url) ? <img className='h-4' src={`/providers/${kind(p.url)}.svg`} /> : <p className='font-bold h-4 m-1'>SSO</p>}
                  {kindCount[kind(p.url)] > 1 && (
                    <div className='text-[10px] -mb-2 text-gray-300'>{p.url}</div>
                  )}
                </div>
              </button>
            ))}
          </div>
          <div className='w-full my-8 relative'>
            <div className='absolute inset-0 flex items-center' aria-hidden='true'>
              <div className='w-full border-t border-gray-800' />
            </div>
            <div className='relative flex justify-center text-sm'>
              <span className='px-8 bg-black text-gray-300'>OR</span>
            </div>
          </div>
        </>
      )}

      <form onSubmit={onSubmit} className='flex flex-col w-full max-w-sm relative'>
        <input
          required
          autoFocus
          type='email'
          placeholder='Email'
          onChange={e => {
            setEmail(e.target.value)
            setError('')
          }}
          className={`bg-purple-100/5 border border-zinc-800 text-sm px-5 mt-2 py-3 rounded-full focus:outline-none focus:ring focus:ring-cyan-600 ${error ? 'border-pink-500' : ''}`}
        />
        <input
          required
          type='password'
          placeholder='Password'
          onChange={e => {
            setPassword(e.target.value)
            setError('')
          }}
          className={`bg-purple-100/5 border border-zinc-800 text-sm px-5 mt-2 py-3 rounded-full focus:outline-none focus:ring focus:ring-cyan-600 ${error ? 'border-pink-500' : ''}`}
        />
        <button disabled={!email || !password} className='bg-gradient-to-tr mt-5 from-indigo-300 to-pink-100 hover:from-indigo-200 hover:to-pink-50 rounded-full p-0.5 my-2 disabled:opacity-30'>
          <div className='bg-black rounded-full text-sm px-4 py-3'>
            Login
          </div>
        </button>
        {error && <p className='absolute -bottom-5 w-full mx-auto text-sm text-pink-500 text-center'>{error}</p>}
      </form>
    </div>
  )
}
